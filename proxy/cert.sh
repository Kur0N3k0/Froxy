openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -subj "/CN=FProxy CA"
openssl genrsa -out cert.key 2048
# generate domain using Froxy CA
# openssl req -new -key cert.key -subj /CN=ltra-cc -addext "subjectAltName = DNS:ltra.cc" | openssl x509 -req -days 1095 -CA ca.crt -CAkey ca.key -CAcreateserial -out ltra-cc.crt