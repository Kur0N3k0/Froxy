package main

import (
	"fmt"
	"froxy/proxy"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
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
						return handleHistory(m)
					case MSG_ISSUE:
						return handleIssue(m)
					case MSG_ADD_MATCH_REPLACE:
						return handleAddMatchReplace(m)
					case MSG_ENABLE_MATCH_REPLACE:
					case MSG_DISABLE_MATCH_REPLACE:
						return handleStatusMatchReplace(m)
					case MSG_DELETE_MATCH_REPLACE:
						return handleDeleteMatchReplace(m)
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
