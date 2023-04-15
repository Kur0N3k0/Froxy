package proxy

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gobwas/ws"
	"github.com/valyala/fasthttp"
)

func genCert(host string) {
	certKeyPem, _ := pem.Decode(root.certkey)
	certKey, err := x509.ParsePKCS8PrivateKey(certKeyPem.Bytes)
	if err != nil {
		clog("ParsePKCS8PrivateKey: " + err.Error())
	}

	clientCert := x509.Certificate{
		SerialNumber: big.NewInt(9505),
		Subject: pkix.Name{
			CommonName: host,
		},
		DNSNames:              []string{host},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	block, _ := pem.Decode(root.cacrt)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		clog("ParseCertificate: " + err.Error())
		clog("ParseCertificate Failure")
		return
	}

	caKeyPem, _ := pem.Decode(root.cakey)
	caKey, err := x509.ParsePKCS8PrivateKey(caKeyPem.Bytes)
	if err != nil {
		clog("ParsePKCS8PrivateKey: " + err.Error())
	}
	cert, err := x509.CreateCertificate(rand.Reader, &clientCert, caCert, certKey.(*rsa.PrivateKey).Public(), caKey)
	if err != nil {
		clog("CreateCertificate: " + err.Error())
		fmt.Println("CreateCertificate Failure")
		return
	}

	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: cert})
	certs[host] = out.Bytes()
}

func DecodeBody(body []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "gzip":
		return fasthttp.AppendGunzipBytes(nil, body)
	case "deflate":
		return fasthttp.AppendInflateBytes(nil, body)
	case "br":
		reader := brotli.NewReader(bytes.NewReader(body))
		return ioutil.ReadAll(reader)
	default:
		return nil, fmt.Errorf("unsupported content encoding: %s", encoding)
	}
}

func DecodeBodyAll(contentEncoding string, body []byte) []byte {
	contentEncoding = strings.ReplaceAll(contentEncoding, " ", "")
	encodings := strings.Split(contentEncoding, ",")
	for _, encoding := range encodings {
		result, err := DecodeBody(body, strings.TrimSpace(encoding))
		if err != nil {
			clog("Error decoding body: " + err.Error())
			break
		}
		body = result
	}
	return body
}

func logRequest(req *http.Request) []byte {
	var bodyCopy bytes.Buffer
	req.Body = ioutil.NopCloser(io.TeeReader(req.Body, &bodyCopy))

	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		clog("Error dumping request: " + err.Error())
		return nil
	}

	req.Body = ioutil.NopCloser(&bodyCopy)

	// log.Println("Request:")
	// log.Println(string(dump))
	return dump
}

func logResponse(res *http.Response) []byte {
	var bodyCopy bytes.Buffer
	res.Body = ioutil.NopCloser(io.TeeReader(res.Body, &bodyCopy))

	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		log.Printf("Error dumping response: %v", err)
		return nil
	}

	res.Body = ioutil.NopCloser(&bodyCopy)

	// log.Println("Response:")
	// log.Println(string(dump))
	return dump
}

func fasthttpRequestToHTTPRequest(req *fasthttp.Request) (*http.Request, error) {
	u := url.URL{
		Scheme:   string(req.URI().Scheme()),
		Host:     string(req.URI().Host()),
		Path:     string(req.URI().Path()),
		RawQuery: string(req.URI().QueryString()),
	}

	body := bytes.NewReader(req.Body())
	httpReq, err := http.NewRequest(string(req.Header.Method()), u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.VisitAll(func(key, value []byte) {
		httpReq.Header.Set(string(key), string(value))
	})

	return httpReq, nil
}

func fasthttpResponseToHTTPResponse(resp *fasthttp.Response) (*http.Response, error) {
	// contentLength := int64(0)
	// if resp.Header.ContentLength() > 0 {
	// 	contentLength = int64(resp.Header.ContentLength())
	// }

	httpResp := &http.Response{
		Status:           string(resp.Header.StatusMessage()),
		StatusCode:       resp.StatusCode(),
		Header:           make(http.Header),
		Body:             ioutil.NopCloser(bytes.NewReader(resp.Body())),
		ContentLength:    int64(len(resp.Body())),
		TransferEncoding: nil,
		Close:            resp.ConnectionClose(),
		Uncompressed:     true,
	}

	resp.Header.VisitAll(func(key, value []byte) {
		httpResp.Header.Set(string(key), string(value))
	})

	httpResp.Proto = string(resp.Header.Protocol())

	proto := string(resp.Header.Protocol())
	parts := strings.SplitN(string(proto), "/", 2)
	if len(parts) == 2 {
		httpResp.Proto = parts[1]
		majMin := strings.SplitN(parts[1], ".", 2)
		if len(majMin) == 2 {
			major, err := strconv.Atoi(majMin[0])
			if err == nil {
				httpResp.ProtoMajor = major
			}
			minor, err := strconv.Atoi(majMin[1])
			if err == nil {
				httpResp.ProtoMinor = minor
			}
		}
	}

	return httpResp, nil
}

func logFrame(prefix string, opcode int, payload []byte) {
	log.Printf("%s OpCode: %d, Payload: %s\n", prefix, opcode, payload)
}

func proxyAndLogWebsocket(client_conn, dest_conn net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			frame, err := ws.ReadFrame(client_conn)
			if err != nil {
				if err != io.EOF {
					log.Println("Error reading frame from client:", err)
				}
				break
			}
			logFrame("Client -> Server", int(frame.Header.OpCode), frame.Payload)

			err = ws.WriteFrame(dest_conn, frame)
			if err != nil {
				log.Println("Error writing frame to destination:", err)
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			frame, err := ws.ReadFrame(dest_conn)
			if err != nil {
				if err != io.EOF {
					log.Println("Error reading frame from destination:", err)
				}
				break
			}
			logFrame("Server -> Client", int(frame.Header.OpCode), frame.Payload)

			err = ws.WriteFrame(client_conn, frame)
			if err != nil {
				log.Println("Error writing frame to client:", err)
				break
			}
		}
	}()

	wg.Wait()
}
