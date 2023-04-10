package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"froxy/proxy"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

var (
	AppName            string
	BuiltAt            string
	VersionAstilectron string
	VersionElectron    string
)

var (
	fs     = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	debug  = fs.Bool("d", false, "enables the debug mode")
	w      *astilectron.Window
	app    *astilectron.Astilectron
	winMgr = map[string]*astilectron.Window{}
)

type MessageType struct {
	MsgType string `json:"type"`
}

type MessageHistoryType struct {
	Idx int `json:"idx"`
}

type MessageIssueType struct {
	Idx     int    `json:"idx"`
	Request string `json:"request"`
	// Response string `json:"response"`
}

const (
	MSG_HISTORY = "history"
	MSG_ISSUE   = "issue"
)

const (
	WIN_MAIN   = "main"
	WIN_FILTER = "filter"
)

func main() {
	if err := bootstrap.Run(bootstrap.Options{
		Asset:    Asset,
		AssetDir: AssetDir,
		AstilectronOptions: astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/app/icons/froxy.icns",
			AppIconDefaultPath: "resources/app/icons/froxy.png",
			SingleInstance:     true,
			VersionAstilectron: VersionAstilectron,
			VersionElectron:    VersionElectron,
		},
		Debug: *debug,
		MenuOptions: []*astilectron.MenuItemOptions{{
			Label: astikit.StrPtr("File"),
			SubMenu: []*astilectron.MenuItemOptions{
				{
					Label: astikit.StrPtr("About"),
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						return
					},
				},
				{
					Accelerator: astilectron.NewAccelerator("CommandOrControl", "C"),
					Role:        astilectron.MenuItemRoleCopy,
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						return
					},
				},
				{
					Accelerator: astilectron.NewAccelerator("CommandOrControl", "V"),
					Role:        astilectron.MenuItemRolePaste,
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						return
					},
				},
				{
					Accelerator: astilectron.NewAccelerator("CommandOrControl", "X"),
					Role:        astilectron.MenuItemRoleCut,
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						return
					},
				},
				{
					Accelerator: astilectron.NewAccelerator("F12"),
					Role:        astilectron.MenuItemRoleToggleDevTools,
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						return
					},
				},
				{Role: astilectron.MenuItemRoleClose},
			},
		}},
		OnWait: func(a *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = ws[0]
			app = a

			winMgr[WIN_MAIN] = w
			w.On(astilectron.EventNameWindowEventClosed, func(e astilectron.Event) (deleteListener bool) {
				delete(winMgr, WIN_MAIN)
				return
			})

			go func() {
				go proxy.RunProxy(":9505", func(history proxy.ProxyHistory) {
					w.SendMessage(map[string]interface{}{
						"type":   "proxy",
						"host":   history.RequestHost,
						"status": history.ResponseStatus,
						"method": history.RequestMethod,
						"url":    history.RequestURL,
						"length": history.ResponseLength,
					})
				}, func(msg string) {
					w.SendMessage(msg)
				})

				w.OnMessage(func(m *astilectron.EventMessage) (v interface{}) {
					var msgType MessageType

					err := m.Unmarshal(&msgType)
					if err != nil {
						w.SendMessage(err.Error())
						return nil
					}

					switch msgType.MsgType {
					case MSG_HISTORY:
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
					case MSG_ISSUE:
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
						// scheme := "http://"
						req.URL.Host = proxy.History[issType.Idx].ServerIp
						req.URL.Scheme = "http"
						if proxy.History[issType.Idx].TLS {
							// scheme = "https://"
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

						encodedBody := ioutil.NopCloser(bytes.NewReader(bbody))
						contentEncoding := res.Header.Get("Content-Encoding")
						var body []byte = bbody
						if contentEncoding != "" {
							encodings := strings.Split(contentEncoding, ",")
							for _, encoding := range encodings {
								decodedBody, err := proxy.DecodeBody(encodedBody, strings.TrimSpace(encoding))
								if err != nil {
									return fmt.Sprintf("Error decoding body: %v\n", err)
								}
								defer decodedBody.Close()

								body, err = ioutil.ReadAll(decodedBody)
								if err != nil {
									return fmt.Sprintf("Error reading decoded body: %v\n", err)
								}
								encodedBody = ioutil.NopCloser(bytes.NewReader(body))
							}
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
					return nil
				})
			}()
			return nil
		},
		RestoreAssets: RestoreAssets,
		Windows: []*bootstrap.Window{{
			Homepage: "home.html",
			// MessageHandler: handleMessage,
			Options: &astilectron.WindowOptions{
				BackgroundColor: astikit.StrPtr("#333"),
				Center:          astikit.BoolPtr(true),
				Height:          astikit.IntPtr(700),
				Width:           astikit.IntPtr(700),
			},
		}},
	}); err != nil {
		fmt.Println("running bootstrap failed: %w", err)
		panic(err)
	}
}
