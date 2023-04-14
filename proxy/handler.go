package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
)

type contextKey struct {
	key string
}

var ConnContextKey = &contextKey{"http-conn"}

func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}
func GetConn(r *http.Request) net.Conn {
	return r.Context().Value(ConnContextKey).(net.Conn)
}

func handleTunneling(ctx *fasthttp.RequestCtx) {
	ctxHost := string(ctx.Host())
	host := strings.Split(ctxHost, ":")[0]
	fmt.Println("Tunneling:", host)

	var dest_conn *tls.Conn
	var err error

	if Socks5 != nil {
		con, err := Socks5.Dial("tcp", ctxHost)
		if err != nil {
			clog("Socks5.Dial: " + err.Error())
			ctx.Error(err.Error(), http.StatusServiceUnavailable)
			return
		}
		dest_conn = tls.Client(con, &tls.Config{InsecureSkipVerify: true})
		dest_conn.Handshake()
	} else {
		dest_conn, err = tls.Dial("tcp", ctxHost, &tls.Config{InsecureSkipVerify: true})
	}

	if err != nil {
		clog("tls.Dial: " + err.Error())
		ctx.Error(err.Error(), http.StatusServiceUnavailable)
		return
	}

	if _, ok := certs[host]; !ok {
		genCert(host)
	}

	cert, err := tls.X509KeyPair(certs[host], root.certkey)
	if err != nil {
		clog("server: loadkeys: " + err.Error())
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Hijack(func(server net.Conn) {
		client_conn := tls.Server(server, &config)
		client_conn.Handshake()

		defer dest_conn.Close()
		defer client_conn.Close()

		clientReader := bufio.NewReader(client_conn)
		destReader := bufio.NewReaderSize(dest_conn, 64*1024)

		for {
			req := fasthttp.AcquireRequest()
			defer fasthttp.ReleaseRequest(req)

			err := req.Read(clientReader)
			if err != nil {
				clog("Error reading from client: " + err.Error())
				return
			}

			body := req.Body()
			contentEncoding := string(req.Header.ContentEncoding())
			if contentEncoding != "" {
				clog(contentEncoding)
				body = DecodeBodyAll(contentEncoding, body)
			}

			req.Header.Del("Transfer-Encoding")
			req.Header.Del("Content-Encoding")

			if len(body) > 0 {
				req.SetBody(body)
				req.Header.SetContentLength(len(body))
			}

			// remove req header hop-by-hop
			req.Header.Del("Connection")
			req.Header.Del("Keep-Alive")
			req.Header.Del("Proxy-Authenticate")
			req.Header.Del("Proxy-Authorization")
			req.Header.Del("TE")
			req.Header.Del("Trailers")
			req.Header.Del("Transfer-Encoding")
			req.Header.Del("Upgrade")

			// ReplaceMatchedRequest(req)
			hreq, err := fasthttpRequestToHTTPRequest(req)
			if err != nil {
				clog("fasthttpRequestToHTTPRequest: " + err.Error())
				return
			}

			rawReq := logRequest(hreq)
			if _, err := req.WriteTo(dest_conn); err != nil {
				clog("Error forwarding request to destination: " + err.Error())
				return
			}

			res := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(res)

			err = res.Read(destReader)
			if err != nil {
				clog("Error reading from server: " + err.Error() + "\n" + string(rawReq))
				return
			}

			body = res.Body()

			contentEncoding = string(res.Header.ContentEncoding())
			if contentEncoding != "" {
				clog(contentEncoding)
				body = DecodeBodyAll(contentEncoding, body)
			}

			res.Header.Del("Transfer-Encoding")
			res.Header.Del("Content-Encoding")

			if len(body) > 0 {
				res.SetBody(body)
				res.Header.SetContentLength(len(body))
			}

			// ReplaceMatchedResponse(res)
			hres, err := fasthttpResponseToHTTPResponse(res)
			if err != nil {
				clog("fasthttpResponseToHTTPResponse: " + err.Error())
				return
			}

			rawRes := logResponse(hres)
			if _, err := res.WriteTo(client_conn); err != nil {
				clog("Error forwarding response to client: " + err.Error())
				return
			}

			hostname := string(ctx.Host())
			history := ProxyHistory{
				ServerIp:       hostname,
				TLS:            true,
				RequestMethod:  hreq.Method,
				RequestHost:    hostname,
				RequestURL:     hreq.URL.RequestURI(),
				ResponseLength: hres.ContentLength,
				ResponseStatus: hres.StatusCode,
				RawRequest:     rawReq,
				RawResponse:    rawRes,
			}
			historyMutex.Lock()
			History = append(History, history)
			historyMutex.Unlock()
			callback(history)
		}
	})
}

func handleHTTP(ctx *fasthttp.RequestCtx) {
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	client := fasthttp.Client{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if Socks5 != nil {
		client.Dial = func(addr string) (net.Conn, error) {
			return Socks5.Dial("tcp", addr)
		}
	}

	ReplaceMatchedRequest(&ctx.Request)
	if err := client.Do(&ctx.Request, resp); err != nil {
		ctx.Error(err.Error(), http.StatusServiceUnavailable)
		return
	}

	contentEncoding := string(ctx.Request.Header.ContentEncoding())
	if contentEncoding != "" {
		clog(contentEncoding)
		result := DecodeBodyAll(contentEncoding, ctx.Request.Body())
		ctx.Request.SetBody(result)
		ctx.Request.Header.SetContentLength(len(result))
	}

	req, err := fasthttpRequestToHTTPRequest(&ctx.Request)
	if err != nil {
		clog(err.Error())
	}

	body := resp.Body()

	contentEncoding = string(resp.Header.Peek("Content-Encoding"))
	if contentEncoding != "" {
		clog(contentEncoding)
		body = DecodeBodyAll(contentEncoding, body)
	}

	resp.Header.Del("Transfer-Encoding")
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp.SetBody(body)

	ReplaceMatchedResponse(resp)
	newResponse, err := fasthttpResponseToHTTPResponse(resp)
	if err != nil {
		clog(err.Error())
	}

	rawReq := logRequest(req)
	rawRes := logResponse(newResponse)

	curhis := ProxyHistory{
		ServerIp:       string(ctx.Host()),
		TLS:            false,
		RequestMethod:  string(ctx.Method()),
		RequestHost:    string(ctx.Host()),
		RequestURL:     ctx.URI().String(),
		ResponseLength: int64(len(body)),
		ResponseStatus: newResponse.StatusCode,
		RawRequest:     rawReq,
		RawResponse:    rawRes,
	}
	historyMutex.Lock()
	History = append(History, curhis)
	callback(curhis)
	historyMutex.Unlock()

	log.Println(req.RemoteAddr, " ", resp.StatusCode())

	ctx.Response.SetStatusCode(resp.StatusCode())
	ctx.Response.Header.SetProtocol(resp.Header.Protocol())
	resp.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.SetBytesKV(key, value)
	})
	ctx.Response.SetBody(resp.Body())
}
