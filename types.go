package main

import (
	"flag"
	"os"

	"github.com/asticode/go-astilectron"
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

type MessageMatchReplaceType struct {
	Idx     int    `json:"idx"`
	Type    string `json:"type"`
	Regex   string `json:"regex"`
	Repl    string `json:"repl"`
	Enabled bool   `json:"enabled"`
}

type MessageSocks5Type struct {
	Addr string `json:"addr"`
}

const (
	MSG_HISTORY               = "history"
	MSG_ISSUE                 = "issue"
	MSG_ADD_MATCH_REPLACE     = "add_match_replace"
	MSG_ENABLE_MATCH_REPLACE  = "enable_match_replace"
	MSG_DISABLE_MATCH_REPLACE = "disable_match_replace"
	MSG_DELETE_MATCH_REPLACE  = "delete_match_replace"
	MSG_SET_SOCKS5            = "set_socks5"
)

const (
	WIN_MAIN   = "main"
	WIN_FILTER = "filter"
)
