#!/usr/bin/env bash
set -euo pipefail

out_dir="${1:-certs}"
mkdir -p "$out_dir"

cat >"$out_dir/server.ext" <<'EOF'
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=@alt_names

[alt_names]
DNS.1=localhost
IP.1=127.0.0.1
EOF

cat >"$out_dir/client.ext" <<'EOF'
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
EOF

openssl genrsa -out "$out_dir/ca.key" 2048
openssl req -x509 -new -nodes -key "$out_dir/ca.key" -sha256 -days 3650 \
  -subj "/CN=Cmdra Dev CA" -out "$out_dir/ca.crt"

openssl genrsa -out "$out_dir/server.key" 2048
openssl req -new -key "$out_dir/server.key" -subj "/CN=localhost" -out "$out_dir/server.csr"
openssl x509 -req -in "$out_dir/server.csr" -CA "$out_dir/ca.crt" -CAkey "$out_dir/ca.key" \
  -CAcreateserial -out "$out_dir/server.crt" -days 365 -sha256 -extfile "$out_dir/server.ext"

for client in client-a client-b; do
  openssl genrsa -out "$out_dir/${client}.key" 2048
  openssl req -new -key "$out_dir/${client}.key" -subj "/CN=${client}" -out "$out_dir/${client}.csr"
  openssl x509 -req -in "$out_dir/${client}.csr" -CA "$out_dir/ca.crt" -CAkey "$out_dir/ca.key" \
    -CAserial "$out_dir/ca.srl" -out "$out_dir/${client}.crt" -days 365 -sha256 -extfile "$out_dir/client.ext"
done

rm -f "$out_dir/"*.csr "$out_dir/"*.ext

printf 'Wrote development certificates to %s\n' "$out_dir"
