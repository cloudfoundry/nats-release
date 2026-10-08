---
title: BOSH Jobs
expires_at: never
tags: [nats-release]
---

# BOSH Jobs

### NATS server

As part of a platform-wide initiative across Cloud Foundry we secure all
internal traffic using TLS. The release provides a single NATS job
(`nats-tls`) serving TLS traffic.

> **NOTE**: NATS does not use a standard TLS over TCP handshake. There is an
> initial INFO handshake, which is via plain-text.  If both the client and the
> server agree in this handshake to use TLS then the connection is upgraded. If
> either of the client or server expects to use TLS but its peer does not then
> they will refuse to connect to avoid downgrade attacks.

#### NATS-TLS

NATS serving TLS traffic.

### smoke-tests

The smoke tests errand run a simple check that NATS is accessible and relaying
messages properly. It will try to use all configured server connections.

## Config Tests
If you add a spec value, please add a corresponding test to
[spec/nats-tls/nats_tls_config_spec.rb](../spec/nats-tls/nats_tls_config_spec.rb)

