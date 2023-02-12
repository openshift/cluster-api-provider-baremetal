#!/usr/bin/env bash
set -euo pipefail

function copy_crd {
  local SRC="$1"
  local DST="$2"
  if ! diff -Naup "$SRC" "$DST"; then
    cp "$SRC" "$DST"
    echo "installed CRD: $SRC => $DST"
  fi
}

REPO_ROOT=$(dirname "${BASH_SOURCE}")/..

tmpfile="$(mktemp tmp-crd-XXXXXXXX)"

# install metal3 remediation crd
for crd in infrastructure.cluster.x-k8s.io_metal3remediations.yaml infrastructure.cluster.x-k8s.io_metal3remediationtemplates.yaml; do
  curl -o $tmpfile https://raw.githubusercontent.com/metal3-io/cluster-api-provider-metal3/main/config/crd/bases/${crd}
  if [ $? -eq 0 ]; then
    copy_crd $tmpfile "config/crd/${crd}"
  fi
done

rm -f $tmpfile
