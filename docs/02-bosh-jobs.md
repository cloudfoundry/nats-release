---
title: BOSH Jobs
expires_at: never
tags: [nats-release]
---

# BOSH Jobs

### NATS server

As part of a platform-wide initiative across Cloud Foundry we are working toward
securing all internal traffic using TLS. NATS servers can support either TLS or
plaint-text traffic, but not both at the same time.
To give clients time to upgrade we are providing two NATS jobs that can be
colocated: a plain-text one (`nats`) and a TLS one (`nats-tls`).

> **NOTE**: NATS does not use a standard TLS over TCP handshake. There is an
> initial INFO handshake, which is via plain-text.  If both the client and the
> server agree in this handshake to use TLS then the connection is upgraded. If
> either of the client or server expects to use TLS but its peer does not then
> they will refuse to connect to avoid downgrade attacks.

#### NATS

NATS serving plain-text traffic. This job will be removed when *all* Cloud
Foundry NATS clients are upgraded to use TLS.

#### NATS-TLS

NATS serving TLS traffic.

### smoke-tests

The smoke tests errand run a simple check that NATS is accessible and relaying
messages properly. It will try to use all configured server connections.

## Config Tests
If you add a spec value, please add a corresponding test to
[spec/nats/nats_config_spec.rb](../spec/nats/nats_config_spec.rb) and
[spec/nats-tls/nats_tls_config_spec.rb](../spec/nats-tls/nats_tls_config_spec.rb)

