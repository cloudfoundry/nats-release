# Welcome to NATS release


This repository contains the nats-release source code.
This is NATS deployed as a BOSH release.
See the [BOSH](http://bosh.io/) documentation for more information on BOSH.

## BOSH Jobs

### NATS server

As part of a platform-wide initiative across Cloud Foundry we are working
toward securing all internal traffic using TLS. NATS servers can support
either TLS or plaint-text traffic,
but not both at the same time <sup>[1](#tlsfootnote)</sup>.
To give clients time to upgrade we are providing two NATS jobs
that can be colocated: a plain-text one (`nats`) and a TLS one (`nats-tls`).

<a name="tlsfootnote">1</a>: NATS does not use a standard TLS over
TCP handshake. There is an initial INFO handshake, which is via plain-text.
If both the client and the server agree in this handshake to use TLS
then the connection is upgraded. If either of the client or server
expects to use TLS but its peer does not then they will refuse to connect
to avoid downgrade attacks.

#### nats

NATS serving plain-text traffic. This job will be removed when
*all* Cloud Foundry NATS clients are upgraded to use TLS.

#### nats-tls

NATS serving TLS traffic.

### smoke-tests

The smoke tests errand run a simple check that NATS is accessible and
relaying messages properly. It will try to use all configured server
connections.

## File a Bug

Bugs can be filed using
[GitHub Issues](http://github.com/cloudfoundry/nats-release/issues/new).
