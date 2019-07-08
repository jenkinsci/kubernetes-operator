FROM alpine:3.9

ENV USER=user
ENV UID=1000
ENV GID=1000

RUN addgroup --gid "$GID" "$USER" && \
    adduser \
    --disabled-password \
    --gecos "" \
    --ingroup "$USER" \
    --uid "$UID" \
    "$USER" && \
    apk add --no-cache bash
WORKDIR /home/user/bin
COPY bin .
RUN chmod +x *.sh
USER user

CMD exec /bin/sh -c "while true; do sleep 1000; done"
