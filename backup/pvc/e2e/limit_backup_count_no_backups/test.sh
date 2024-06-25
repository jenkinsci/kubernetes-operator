#!/bin/bash
set -eo pipefail

echo "Running limit_backup_count_no_backups e2e test..."

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
mkdir -p ${JENKINS_HOME}

# Create an instance of the container under testing
cid="$(docker run -e BACKUP_CLEANUP_INTERVAL=1 -e BACKUP_COUNT=2 -e JENKINS_HOME=${JENKINS_HOME} -v ${JENKINS_HOME}:${JENKINS_HOME}:ro -e BACKUP_DIR=${BACKUP_DIR} -v ${BACKUP_DIR}:${BACKUP_DIR}:rw -d ${docker_image})"
echo "Docker container ID '${cid}'"

# Remove test directory and container afterwards
trap "docker rm -vf $cid > /dev/null;rm -rf ${BACKUP_DIR};rm -rf ${JENKINS_HOME}" EXIT

# container should be running
echo 'Checking if container is running'
sleep 3
set +e
docker exec ${cid} echo
exit_code=$?
set -e
if [ $exit_code -ne 0 ]; then
    echo "container terminated with following logs:"
    docker logs "${cid}"
    exit 1
fi
echo 'Container is running'

echo PASS
