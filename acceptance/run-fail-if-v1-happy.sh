#!/bin/bash

set  -x
TESTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
RELEASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd ../ &&  pwd )"

echo "-----> `date`: Delete previous deployment"
bosh -n -d nats delete-deployment --force

echo "-----> `date`: Deploy dev release"
( set -e;  bosh -n -d nats deploy  $TESTDIR/manifest-non-tls.yml -o $TESTDIR/replace-with-dev.yml -o $TESTDIR/properties-fail-if-v1.yml -o $TESTDIR/100-max-flight.yml )

if [[ $? == 1 ]]; then
    echo "Deployment failed unexpectedlly. Failing."
    exit 1
else
    echo "Deployment succeeded."
fi

echo "-----> `date`: Checking results"
bosh -d nats ssh -c "cd /var/vcap/sys/log/nats-tls && sudo tail post-start.stdout.log | grep 'Local nats server version: 2'"
if [[ $? == 0 ]]; then
    echo "V2 confirmation logged as expected."
else
    echo "No v2 confirmation message. Failing."
    exit 1
fi

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
