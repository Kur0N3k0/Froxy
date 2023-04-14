package proxy

import (
	"regexp"
	"sync"
)

type rootCert struct {
	certkey []byte
	cacrt   []byte
	cakey   []byte
}

type ProxyHistory struct {
	ServerIp       string
	TLS            bool
	RequestMethod  string
	RequestHost    string
	RequestURL     string
	ResponseLength int64
	ResponseStatus int
	RawRequest     []byte
	RawResponse    []byte
}

const (
	MR_REQUEST_HEADER  = "REQUEST_HEADER"
	MR_REQUEST_BODY    = "REQUEST_BODY"
	MR_RESPONSE_HEADER = "RESPONSE_HEADER"
	MR_RESPONSE_BODY   = "RESPONSE_BODY"
)

type RegexMatchReplace struct {
	Type    string
	Regex   *regexp.Regexp
	Replace string
	Enabled bool
}

var root rootCert
var certs map[string][]byte = make(map[string][]byte)
var callback func(ProxyHistory)
var clog func(string)
var History []ProxyHistory = make([]ProxyHistory, 0)
var historyMutex sync.Mutex
