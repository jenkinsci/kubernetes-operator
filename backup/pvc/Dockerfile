FROM debian:buster-slim

ENV USER=user
ENV UID=1000
ENV GID=1000

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
