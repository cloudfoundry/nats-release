#!/bin/bash

set  -x
TESTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
RELEASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd ../ &&  pwd )"

echo "-----> `date`: Delete previous deployment"
bosh -n -d nats delete-deployment --force

echo "-----> `date`: Deploy with nats v1"
( set -e;  bosh -n -d nats deploy $TESTDIR/manifest-non-tls.yml )

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

echo "-----> `date`: Modifying dev release to force v1 "
( cp $TESTDIR/nats-tls/migrator-config.json.erb $RELEASEDIR/jobs/nats-tls/templates/migrator-config.json.erb )
( cp $TESTDIR/nats/migrator-config.json.erb $RELEASEDIR/jobs/nats/templates/migrator-config.json.erb )


echo "-----> `date`: Deploy dev release"
( set -e;  bosh -n -d nats deploy  $TESTDIR/manifest-non-tls.yml -o $TESTDIR/replace-with-dev.yml -o $TESTDIR/properties-fail-if-v1.yml -o $TESTDIR/100-max-flight.yml )

if [[ $? == 0 ]]; then
    echo "Deployment should fail with v1. Failing."
    exit 1
else
    echo "Deployment failed as expected."
fi


echo "-----> `date`: Checking results"
bosh -d nats ssh nats/0 -c "cd /var/vcap/sys/log/nats-tls && sudo tail post-start.stdout.log | grep 'Local NATS server is on v1; exiting with error'"
if [[ $? == 0 ]]; then
    echo "V1 failure message logged as expected."
else
    echo "No v1 failure message."
    exit 1
fi


bosh -d nats ssh nats/1 -c "cd /var/vcap/sys/log/nats-tls && sudo tail post-start.stdout.log | grep 'Skipping because instance is not canary'"
if [[ $? == 0 ]]; then
    echo "Non-canary message logged as expected."
else
    echo "No non-canary message."
    exit 1
fi

bosh -d nats ssh -c "ps aux | grep -v grep | grep nats-server"
if [[ $? == 0 ]]; then
    echo "NATS v2 incorrectly running after failed deployment. Fail"
    exit 1
else
    echo "No NATS v2."
fi
bosh -d nats ssh -c "ps aux | grep -v grep | grep gnatsd"
if [[ $? == 0 ]]; then
    echo "NATS v1 correctly running after failed deployment."
else
    echo "No NATS v1."
    exit 1
fi

echo "-----> `date`: Undoing changes to dev release "
( cd $RELEASEDIR && git checkout -- . )
echo "-----> `date`: Done"
