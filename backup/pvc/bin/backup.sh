#!/usr/bin/env bash

set -eo pipefail

[[ ! $# -eq 1 ]] && echo "Usage: $0 backup_number" && exit 1;
[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" && exit 1;
[[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;

# create temp dir on the same filesystem with a BACKUP_DIR to be able use atomic mv enstead of copy
BACKUP_TMP_DIR=$(mktemp -d --tmpdir=${BACKUP_DIR})
trap "test -d "${BACKUP_TMP_DIR}" && rm -fr "${BACKUP_TMP_DIR}"" EXIT SIGINT SIGTERM

backup_number=$1
echo "Running backup"

# config.xml in a job directory is a config file that shouldn't be backed up
# config.xml in child directories is state that should. For example-
# branches/myorg/branches/myrepo/branches/master/config.xml should be retained while
# branches/myorg/config.xml should not
tar --zstd -C "${JENKINS_HOME}" -cf "${BACKUP_TMP_DIR}/${backup_number}.tar.zstd" \
    --exclude jobs/*/workspace* \
    --no-wildcards-match-slash --anchored \
    --ignore-failed-read \
    --exclude jobs/*/config.xml -c jobs || ret=$?

if [[ "$ret" -eq 0 ]]; then
  echo "Backup was completed without warnings"
elif [[ "$ret" -eq 1 ]]; then
  echo "Backup was completed with some warnings"
fi

# atomically create a backup file
mv "${BACKUP_TMP_DIR}/${backup_number}.tar.zstd" "${BACKUP_DIR}/${backup_number}.tar.zstd"

rm -rf "${BACKUP_TMP_DIR}"
[[ ! -s ${BACKUP_DIR}/${backup_number}.tar.zstd ]] && echo "backup file '${BACKUP_DIR}/${backup_number}.tar.zstd' is empty" && exit 1;

echo Done
exit 0
