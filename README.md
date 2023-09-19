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

## Config Tests
If you add a spec value, please add a corresponding test to
[spec/nats/nats_config_spec.rb](spec/nats/nats_config_spec.rb) and
[spec/nats-tls/nats_tls_config_spec.rb](spec/nats-tls/nats_tls_config_spec.rb)

### <a name="developer-workflow"></a> Developer Workflow

1. Clone CI repository (next to where nats-release is cloned)

  ```bash
  mkdir -p ~/workspace
  cd ~/workspace
  git clone https://github.com/cloudfoundry/wg-app-platform-runtime-ci.git
  ```

#### <a name="running-unit-and-integration-tests"></a> Running Unit and Integration Tests

##### With Docker

Running tests for this release requires Linux specific setup and it takes advantage of having the same configuration as Concourse CI, so it's recommended to run the tests (units & integration) in docker containers.

1. `./scripts/docker/container.bash`: This will create a docker container (cloudfoundry/tas-runtime-build) with appropriate mounts.

- `/repo/scripts/docker/test.bash`: This will run all tests in this release
- `/repo/scripts/docker/tests-templates.bash`: This will run all of tests for bosh tempalates
- `/repo/scripts/docker/lint.bash`: This will run all of linting defined for this repo.

There are also these scripts to make local development/testing easier:
- `./scripts/test-in-docker-locally`: Runs template tests, the test.bash script, and linters.

## File a Bug

Bugs can be filed using
[GitHub Issues](http://github.com/cloudfoundry/nats-release/issues/new).
