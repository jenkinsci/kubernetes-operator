#!/bin/bash
set -eo pipefail

echo "Running stale_tmp_dir_cleanup e2e test..."

[[ "${DEBUG}" ]] && set -x

# set current working directory to the directory of the script
cd "$(dirname "$0")"

docker_image=$1

if ! docker inspect ${docker_image} &> /dev/null; then
    echo "Image '${docker_image}' does not exists"
    false
fi

JENKINS_HOME="$(pwd)/jenkins_home"
BACKUP_DIR="$(pwd)/backup"
mkdir -p ${BACKUP_DIR}

# Create stale directories that should be cleaned up
mkdir -p ${BACKUP_DIR}/tmp.stale1
mkdir -p ${BACKUP_DIR}/tmp.stale2
touch ${BACKUP_DIR}/tmp.stale1/somefile

# Create directories that should NOT be cleaned up
mkdir -p ${BACKUP_DIR}/other_dir
mkdir -p ${BACKUP_DIR}/tmp_but_no_dot
mkdir -p ${BACKUP_DIR}/tmp
touch ${BACKUP_DIR}/other_dir/keepme

# Create an instance of the container under testing
cid="$(docker run -e BACKUP_CLEANUP_INTERVAL=1 -e JENKINS_HOME=${JENKINS_HOME} -v ${JENKINS_HOME}:${JENKINS_HOME}:ro -e BACKUP_DIR=${BACKUP_DIR} -v ${BACKUP_DIR}:${BACKUP_DIR}:rw -d ${docker_image})"
echo "Docker container ID '${cid}'"

# Remove test directory and container afterwards
trap "docker rm -vf $cid > /dev/null;rm -rf ${BACKUP_DIR}" EXIT

backup_number=1
docker exec ${cid} /home/user/bin/backup.sh ${backup_number}

# Check cleanup results
if [ -d "${BACKUP_DIR}/tmp.stale1" ]; then
    echo "FAIL: Stale directory tmp.stale1 was not removed"
    exit 1
fi

if [ -d "${BACKUP_DIR}/tmp.stale2" ]; then
    echo "FAIL: Stale directory tmp.stale2 was not removed"
    exit 1
fi

if [ ! -d "${BACKUP_DIR}/other_dir" ]; then
    echo "FAIL: Directory other_dir was incorrectly removed"
    exit 1
fi

if [ ! -d "${BACKUP_DIR}/tmp_but_no_dot" ]; then
    echo "FAIL: Directory tmp_but_no_dot was incorrectly removed"
    exit 1
fi

if [ ! -d "${BACKUP_DIR}/tmp" ]; then
    echo "FAIL: Directory tmp was incorrectly removed"
    exit 1
fi

# Verify backup success
backup_file="${BACKUP_DIR}/${backup_number}.tar.zstd"
[[ ! -f ${backup_file} ]] && echo "Backup file ${backup_file} not found" && exit 1;

# Verify no unexpected tmp directories remain
remaining_tmp=$(find "${BACKUP_DIR}" -maxdepth 1 -name "tmp.*")
if [ ! -z "$remaining_tmp" ]; then
    echo "FAIL: Unexpected tmp directories remaining: $remaining_tmp"
    exit 1
fi

echo "Stale temporary directories cleaned up successfully"
echo PASS
