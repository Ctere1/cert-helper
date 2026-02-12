# Using cert-helper for EAP-TLS & IAM Testing

This document explains **why** the cert-helper project exists and how it should be used while building and validating IAM capabilities that issue certificates for Wi‑Fi (EAP‑TLS) and similar mutual‑TLS scenarios.

The short version:

> cert-helper is a controlled PKI laboratory for modelling enterprise trust: offline root, operational intermediate, and correctly profiled client/server certificates.

---

## Goals of the Repository

When developing IAM features, teams repeatedly need to:

* bootstrap a trust hierarchy
* experiment with certificate profiles
* validate EKU / KeyUsage combinations
* reproduce customer environments
* debug RADIUS or supplicant behaviour

Doing this work with production CAs is risky, slow, and often impossible.

cert-helper provides a **repeatable and disposable environment** that behaves like a real PKI without operational impact.

---

## Target Architecture (What We Try to Simulate)

```
           Root CA (offline in real life)
           /                     \
          /                       \
   Intermediate CA (inside IAM)   RADIUS Server
          |
          |
       Client Certificates
```

### Trust chain used during authentication

```
client → intermediate → root
```

---

## How cert-helper Maps to the IAM World

| Real System            | cert-helper role                 |
| ---------------------- | -------------------------------- |
| Offline root CA        | Root CA generated once           |
| IAM signing authority  | Intermediate CA                  |
| User/device enrollment | End-entity certificate issuance  |
| RADIUS trust store     | root + intermediate certificates |

---

## What We Validate With It

Typical questions answered via this repository:

* Does the client certificate contain **Client Authentication EKU**?
* Will RADIUS build and trust the chain?
* What happens if ServerAuth is also present?
* Which SAN format works best for username extraction?
* How do different key algorithms affect compatibility?

---

## EAP-TLS Critical Requirements Reminder

### Client certificates must include

* Key Usage → Digital Signature
* EKU → TLS Web Client Authentication (1.3.6.1.5.5.7.3.2)

Without these, most RADIUS implementations will reject authentication.

### Server certificates must include

* EKU → TLS Web Server Authentication

---

## Why Intermediate CA Matters

In real deployments, the root private key is never stored inside IAM or any online system.

Instead:

* the Root CA signs the intermediate,
* IAM uses the intermediate for day‑to‑day issuance.

If IAM is compromised → rotate the intermediate, not the root.

cert-helper allows teams to practice and automate this lifecycle.

---

## Algorithm Experiments

The toolkit is also useful for validating choices such as:

* RSA vs ECDSA
* P256 vs P384
* TLS handshake performance
* legacy device behaviour

---

## Non‑Goals

cert-helper is **not** a production CA. It is a development and validation utility and should never be used to protect real identities or infrastructure.
