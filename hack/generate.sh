#!/bin/sh

# Ignore the rule that says we should always quote variables, because
# in this script we *do* want globbing.
# shellcheck disable=SC2086

set -e

ARTIFACTS=${ARTIFACTS:-/tmp}

eval "$(go env)"
#cd "${GOPATH}"/src/github.com/openshift/cluster-api-provider-baremetal
export XDG_CACHE_HOME="/tmp/.cache"

INPUT_FILES="pkg/apis/baremetal/v1alpha1//zz_generated.*.go"
cksum $INPUT_FILES > "$ARTIFACTS/lint.cksums.before"
make generate
cksum $INPUT_FILES > "$ARTIFACTS/lint.cksums.after"
diff "$ARTIFACTS/lint.cksums.before" "$ARTIFACTS/lint.cksums.after"
