#!/bin/bash

ca_cert_file="./proxy/ca.crt"
ca_key_file="./proxy/ca.key"
cert_key_file="./proxy/cert.key"
proxy_template_file="./proxy/proxy_go.tmpl"
output_file="./proxy/froxy.go"

if [ ! -f "proxy/ca.key" ] || [ ! -f "proxy/ca.crt" ] || [ ! -f "proxy/cert.key" ]; then
    cd ./proxy
    ./cert.sh
    cd ..
fi

ca_cert=$(cat "$ca_cert_file")
ca_key=$(cat "$ca_key_file")
cert_key=$(cat "$cert_key_file")

proxy_template=$(cat "$proxy_template_file")
proxy_go="${proxy_template//__CA_CRT__/$ca_cert}"
proxy_go="${proxy_go//__CA_KEY__/$ca_key}"
proxy_go="${proxy_go//__CERT_KEY__/$cert_key}"

echo "$proxy_go" > "$output_file"

astilectron-bundler