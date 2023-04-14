package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"froxy/proxy"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/asticode/go-astilectron"
)

func handleHistory(m *astilectron.EventMessage) interface{} {
	var hisType MessageHistoryType
	m.Unmarshal(&hisType)
	if hisType.Idx < 0 || len(proxy.History) <= hisType.Idx {
		return map[string]interface{}{
			"type": "error",
		}
	}
	return map[string]interface{}{
		"type":     "history",
		"request":  proxy.History[hisType.Idx].RawRequest,
		"response": proxy.History[hisType.Idx].RawResponse,
	}
}

func handleIssue(m *astilectron.EventMessage) interface{} {
	var issType MessageIssueType
	m.Unmarshal(&issType)
	if issType.Idx < 0 || len(proxy.History) <= issType.Idx {
		return map[string]interface{}{
			"type": "error",
		}
	}

	w.SendMessage(proxy.History[issType.Idx].ServerIp)
	w.SendMessage(issType.Request)
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(issType.Request)))
	if err != nil {
		return "ReadRequest " + err.Error()
	}
	w.SendMessage(fmt.Sprintf("%v", req))
	req.URL.Host = proxy.History[issType.Idx].ServerIp
	req.URL.Scheme = "http"
	if proxy.History[issType.Idx].TLS {
		req.URL.Scheme = "https"
	}

	tbody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.SendMessage(err.Error())
	}
	nreq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(tbody))
	if err != nil {
		return "NewRequest " + err.Error()
	}
	nreq.Proto = req.Proto
	nreq.Header = req.Header.Clone()
	nreq.Header.Set("Content-Length", strconv.Itoa(len(tbody)))

	kProxyUrl, _ := url.Parse("http://127.0.0.1:9505")
	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
			Proxy:              http.ProxyURL(kProxyUrl),
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Do(nreq)
	if err != nil {
		return "Do " + err.Error()
	}
	defer res.Body.Close()

	bbody, _ := ioutil.ReadAll(res.Body)

	contentEncoding := res.Header.Get("Content-Encoding")
	var body []byte = bbody
	if contentEncoding != "" {
		body = proxy.DecodeBodyAll(contentEncoding, body)
	}

	delete(res.Header, "Transfer-Encoding")
	delete(res.Header, "Content-Encoding")

	newResponse := &http.Response{
		Status:           res.Status,
		StatusCode:       res.StatusCode,
		Proto:            res.Proto,
		ProtoMajor:       res.ProtoMajor,
		ProtoMinor:       res.ProtoMinor,
		Header:           res.Header,
		Body:             ioutil.NopCloser(bytes.NewReader(body)),
		ContentLength:    int64(len(body)),
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     true,
		Request:          res.Request,
		TLS:              res.TLS,
	}

	var bres bytes.Buffer
	writer := bufio.NewWriter(&bres)
	newResponse.Write(writer)
	writer.Flush()

	return map[string]interface{}{
		"type":     "issue",
		"response": bres.Bytes(),
	}
}

func handleAddMatchReplace(m *astilectron.EventMessage) interface{} {
	var mrType MessageMatchReplaceType
	m.Unmarshal(&mrType)

	if regex, err := regexp.Compile(mrType.Regex); err == nil {
		proxy.MrRules = append(proxy.MrRules, proxy.RegexMatchReplace{
			Type:    mrType.Type,
			Regex:   regex,
			Replace: mrType.Repl,
			Enabled: true,
		})
		return map[string]interface{}{
			"type":  "add_match_replace",
			"error": false,
		}
	}

	return map[string]interface{}{
		"type":  "add_match_replace",
		"error": true,
	}
}

func handleStatusMatchReplace(m *astilectron.EventMessage) interface{} {
	var mrType MessageMatchReplaceType
	m.Unmarshal(&mrType)

	if mrType.Idx < 0 || len(proxy.MrRules) <= mrType.Idx {
		return map[string]interface{}{
			"type": "error",
		}
	}

	proxy.MrRules[mrType.Idx].Enabled = mrType.Enabled
	return map[string]interface{}{
		"type": "enable_match_replace",
	}
}

func handleDeleteMatchReplace(m *astilectron.EventMessage) interface{} {
	var mrType MessageMatchReplaceType
	m.Unmarshal(&mrType)

	if mrType.Idx < 0 || len(proxy.MrRules) <= mrType.Idx {
		return map[string]interface{}{
			"type":  "error",
			"error": true,
		}
	}

	proxy.MrRules = append(proxy.MrRules[:mrType.Idx], proxy.MrRules[mrType.Idx+1:]...)
	return map[string]interface{}{
		"type":  "delete_match_replace",
		"error": false,
	}
}
