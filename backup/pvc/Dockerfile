FROM debian:bullseye-slim

LABEL maintainer="Jenkins Kubernetes Operator Community" \
      org.opencontainers.image.authors="Jenkins Kubernetes Operator Community" \
      org.opencontainers.image.title="backup-pvc" \
      org.opencontainers.image.description="Jenkins Operator Backup img via pvc volume" \
      org.opencontainers.image.url="quay.io/jenkins-kubernetes-operator/backup-pvc" \
      org.opencontainers.image.source="https://github.com/jenkinsci/kubernetes-operator/tree/master/backup/pvc" \
      org.opencontainers.image.base.name="debian:bullseye-slim"

ARG UID
ARG GID

ENV USER=user

RUN addgroup --gid "$GID" "$USER" && \
    adduser \
    --disabled-password \
    --gecos "" \
    --ingroup "$USER" \
    --uid "$UID" \
    "$USER"

WORKDIR /home/user/bin
COPY bin .
RUN chmod +x *.sh
USER user

CMD ./run.sh
