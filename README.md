# cert-helper

`cert-helper` is a lightweight certificate toolkit that helps teams generate root CAs, intermediate CAs, and end-entity certificates via both CLI and a web dashboard.

## Why this project exists

While building IAM capabilities, teams need a controllable PKI playground to understand how certificates behave in real customer environments for EAP-TLS.

**cert-helper** provides that sandbox.

It enables simulation of an enterprise architecture where:

* the Root CA is offline and heavily protected,
* an Intermediate CA inside IAM performs day-to-day signing,
* RADIUS or other gateways validate identities through the trust chain,
* and certificate profiles (EKU, SAN, KeyUsage, algorithms) can be tuned, validated, and iterated safely.

For deeper background and architectural guidance, see [eap-tls-usage.md](eap-tls-usage.md).

## Features
- Root CA generation (self-signed)
- Intermediate CA generation signed by a selected root CA
- End-entity certificate generation with SANs and PFX output
- Web dashboard to create, browse, and download generated certificates

## Requirements
- Go 1.20+

## CLI Usage

### Root CA
```bash
go run main.go ca generate \
  --name default \
  --common-name "Example Root CA" \
  --organization "Example Org"
```

### Intermediate CA
```bash
go run main.go ca intermediate \
  --root default \
  --name "Example Intermediate" \
  --common-name "Example Intermediate CA"
```

### End-entity certificate
```bash
go run main.go cert generate \
  --issuer-type intermediate \
  --issuer-root default \
  --issuer-name "Example Intermediate" \
  --common-name "api.example.com" \
  --subject-alt-names "api.example.com,api.internal"
```

Subject fields are available on both CA and certificate commands:
- `--common-name` (CN)
- `--organization` (O)
- `--organizational-unit` (OU)
- `--country` (C)
- `--state` (ST)
- `--locality` (L)

## Web Dashboard

```bash
go run main.go serve --output-dir /tmp/cert-helper
```

Open `http://localhost:8000` to generate certificates, view existing CAs, and download files. The file browser is available in the File Center tab at `http://localhost:8000/#files`. Use `--host 0.0.0.0` to expose the dashboard to your network, and do so carefully because the dashboard allows access to generated certificate files and does not include authentication.

## Output Layout

```
output-dir/
  ca.pem / ca.key                    # default root CA
  ca/root/<name>/ca.pem / ca.key     # named root CAs
  ca/intermediate/<root>/<name>/...  # intermediate CAs
  certs/root/<root>/...              # certificates signed by root CA
  certs/intermediate/<root>/<name>/... # certificates signed by intermediate CA
```
