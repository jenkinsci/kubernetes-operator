#!/usr/bin/env bash

set -eo pipefail

[[ ! "${#}" -eq 1 ]] && echo "Usage: ${0} backup_number" >&2 && exit 1;
[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" >&2 && exit 1;
[[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" >&2 && exit 1;

BACKUP_TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${BACKUP_TMP_DIR}"' EXIT

backup_number="${1}"
echo "Running backup"

# config.xml in a job directory is a config file that shouldnt be backed up
# config.xml in child directores is state that should. For example-
# branches/myorg/branches/myrepo/branches/master/config.xml should be retained while
# branches/myorg/config.xml should not
tar \
    --directory="${JENKINS_HOME}" \
    --create \
    --gzip \
    --file "${BACKUP_TMP_DIR}/${backup_number}.tar.gz" \
    --no-wildcards-match-slash \
    --anchored \
    --exclude jobs/*/workspace* \
    --exclude jobs/*/config.xml \
    jobs

mv "${BACKUP_TMP_DIR}/${backup_number}.tar.gz" "${BACKUP_DIR}/${backup_number}.tar.gz"

if [[ ! -s "${BACKUP_DIR}/${backup_number}.tar.gz" ]] ; then
    echo "backup file '${BACKUP_DIR}/${backup_number}.tar.gz' is empty" >&2
    exit 1
fi

echo Done
