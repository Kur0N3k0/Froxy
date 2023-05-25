package main

import (
	"fmt"
	"froxy/proxy"
)

func mainx() {
	proxy.RunProxy(":9505", func(ph proxy.ProxyHistory) {}, func(s string) {
		fmt.Println(s)
	})
}
