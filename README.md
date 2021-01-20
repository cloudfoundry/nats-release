# Welcome to NATS release


This repository contains the nats-release source code. This is NATS deployed as
a BOSH release. See the [BOSH](http://bosh.io/) documentation for more
information on BOSH.

## BOSH Jobs

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

## Example Manifests

In the `example-manifests` folder we have an example of deployment of `nats` and
`nats tls` with the smoke test.

Also, there is an ops file called `enable_nats_tl_for_cf.yml` which can be used
to add `nats` and `nats-tls` jobs with the `smoke-test` errand to the CF
deployment.


## Config Tests
If you add a spec value, please add a corresponding test to
[spec/nats/nats_config_spec.rb](spec/nats/nats_config_spec.rb) and
[spec/nats-tls/nats_tls_config_spec.rb](spec/nats-tls/nats_tls_config_spec.rb)

To run these tests:
```
./scripts/docker_run_job_property_tests
```
## File a Bug

Bugs can be filed using
[GitHub Issues](http://github.com/cloudfoundry/nats-release/issues/new).
