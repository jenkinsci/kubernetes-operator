ARG GO_VERSION
FROM golang:${GO_VERSION} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY test/e2e/testdata/reconciliation-ignore test/e2e/testdata/reconciliation-ignore
RUN CGO_ENABLED=0 go build -o /identity-webhook ./test/e2e/testdata/reconciliation-ignore

FROM scratch
COPY --from=build /identity-webhook /identity-webhook
USER 65532:65532
ENTRYPOINT ["/identity-webhook"]
