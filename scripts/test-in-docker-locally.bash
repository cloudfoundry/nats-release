#!/bin/bash

set -eu

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/git-helpers.bash"
REPO_NAME=$(git_get_remote_name)

"${THIS_FILE_DIR}/create-docker-container.bash" -d

docker exec $REPO_NAME-docker-container '/repo/scripts/docker/tests-templates.bash'
docker exec $REPO_NAME-docker-container '/repo/scripts/docker/test.bash' "$@"
docker exec $REPO_NAME-docker-container '/repo/scripts/docker/lint.bash'
