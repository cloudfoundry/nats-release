#!/bin/bash -exu

set -eu
set -o pipefail

/ci/shared/tasks/run-tests-templates/task.bash
