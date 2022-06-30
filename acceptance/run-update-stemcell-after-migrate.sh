#!/bin/bash

set  -x
TESTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

bosh upload-stemcell https://storage.googleapis.com/bosh-gce-light-stemcells/1.85/light-bosh-stemcell-1.85-google-kvm-ubuntu-bionic-go_agent.tgz

echo "-----> `date`: Delete previous deployment"
bosh -n -d nats delete-deployment --force

echo "-----> `date`: Deploy with nats v1"
( set -e;  bosh -n -d nats deploy $TESTDIR/manifest.yml -o $TESTDIR/use-stemcell-85.yml )

bosh -d nats ssh -c "ps aux | grep -v grep | grep gnats"
if [[ $? == 0 ]]; then
    echo "NATS v1 running before migration."
else
    echo "No NATS v1 before migration. Fail."
    exit 1
fi

bosh -d nats ssh -c "ps aux | grep -v grep | grep nats-server"
if [[ $? == 0 ]]; then
    echo "NATS v2 running before migration. Fail."
    exit 1
else
    echo "No NATS v2"
fi

echo "-----> `date`: Deploy migration release"
( set -e;  bosh -n -d nats deploy $TESTDIR/manifest.yml -o $TESTDIR/use-stemcell-85.yml  -o $TESTDIR/replace-with-dev.yml )


bosh -d nats ssh -c "ps aux | grep -v grep | grep gnats"
if [[ $? == 0 ]]; then
    echo "NATS v1 running after deployment. Fail"
    exit 1
else
    echo "No NATS v1"
fi

bosh -d nats ssh -c "ps aux | grep -v grep | grep nats-server"
if [[ $? == 0 ]]; then
    echo "NATS v2 running after deployment."
else
    echo "No NATS v2"
    exit 1
fi

echo "-----> `date`: Update stemcell"
( set -e;  bosh -n -d nats deploy $TESTDIR/manifest.yml -o $TESTDIR/replace-with-dev.yml )


bosh -d nats ssh -c "ps aux | grep -v grep | grep gnats"
if [[ $? == 0 ]]; then
    echo "NATS v1 running after deployment. Fail"
    exit 1
else
    echo "No NATS v1"
fi

bosh -d nats ssh -c "ps aux | grep -v grep | grep nats-server"
if [[ $? == 0 ]]; then
    echo "NATS v2 running after deployment."
else
    echo "No NATS v2"
    exit 1
fi
echo "-----> `date`: Done"
