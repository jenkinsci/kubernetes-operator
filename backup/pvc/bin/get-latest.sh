#!/usr/bin/env bash

set -eo pipefail

[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" && exit 1

latest=$(find ${BACKUP_DIR} -name '*.tar.zstd' -exec basename {} \; | sort -g | tail -n 1)

if [[ "${latest}" == "" ]]; then
  echo "-1"
else
  echo "${latest%%.*}"
fi
