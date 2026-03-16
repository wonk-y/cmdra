---
sidebar_position: 1
---

# Generate Development Certificates

Cmdra requires mTLS. The daemon presents a server certificate, and each connecting client presents its own client certificate.

For local development, generate a small CA plus one server certificate and development client certificates:

```bash
./scripts/generate-dev-certs.sh dev/certs
```

This creates:

- `dev/certs/ca.crt`
- `dev/certs/server.crt`
- `dev/certs/server.key`
- `dev/certs/client-a.crt`
- `dev/certs/client-a.key`
- `dev/certs/client-b.crt`
- `dev/certs/client-b.key`

## What matters for validation

- Normal operation only requires one client certificate plus the server certificate.
- The extra `client-b` certificate is generated for authorization and cross-identity testing.
- The server certificate should include DNS/IP SANs for the hostname or IP clients connect to when clients perform normal certificate verification.
- Client SANs are not required for this project because daemon-side authorization is keyed off certificate CN.
- Access control is configured with `--allowed-client-cn`.

## Minimal development layout

```text
dev/certs/
  ca.crt
  server.crt
  server.key
  client-a.crt
  client-a.key
  client-b.crt
  client-b.key
```
