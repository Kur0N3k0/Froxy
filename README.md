# Fast Proxy (HTTP/HTTPS 1.1 Supported)
## Build
- Install DEPS
```bash
# DEPS: go-astilectron-bundler
go get -u github.com/asticode/go-astilectron-bundler/...
go install github.com/asticode/go-astilectron-bundler/astilectron-bundler
```

- Setup Bundler
```json
# bundler.json
{
  "app_name": "Froxy",
  "environments": [
    {"arch": "arm64", "os": "darwin"},
    {"arch": "amd64", "os": "linux"},
    {"arch": "amd64", "os": "windows"}
  ]
}
```

- Build Froxy
```bash
./build.sh
```

## Usage
### Shortcuts
```
F12: Devtools
Ctrl + G: Issue Request (Content-Length auto modified)
Ctrl + B: Selection base64 encoding
Ctrl + Shift + B: Selection base64 decoding
Ctrl + E: Selection URL encoding (encodeURIComponent)
Ctrl + Shift + E: Selection URL decoding (encodeURIComponent)
```

## TODO
- [x] HTTP/HTTPS Proxy
- [x] Shortcuts
- [ ] Request/Response Match & Replace
- [ ] Interceptor
- [ ] En/Decoder Window
- [ ] Socks5 Support
- [ ] Websocket Support
- [ ] HTTP/2 Support
- [ ] Raw TCP Support
- [ ] Encode/Encrypted TCP Packet Plugin(javscript)
- [ ] Hex Editor