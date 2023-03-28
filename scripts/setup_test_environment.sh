#!/bin/bash
export BIN_DIR="$PWD/bin"
export PATH=$PATH:$BIN_DIR

pushd src/code.cloudfoundry.org
    go build -o "$BIN_DIR/gnatsd" github.com/nats-io/gnatsd
    go build -o "$BIN_DIR/nats-server" github.com/nats-io/nats-server/v2
    if ! [ -x "$(command -v ginkgo)" ]; then
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
    fi
popd

echo "Done setting up for tests"
