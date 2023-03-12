ARG GO_VERSION

# Build the manager binary
FROM golang:$GO_VERSION as builder
ARG CTIMEVAR
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY controllers/ controllers/
COPY internal/ internal/
COPY pkg/ pkg/
COPY version/ version/
COPY main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags "-w $CTIMEVAR" -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL maintainer="Jenkins Kubernetes Operator Community" \
      org.opencontainers.image.authors="Jenkins Kubernetes Operator Community" \
      org.opencontainers.image.title="jenkins-kubernetes-operator" \
      org.opencontainers.image.description="Kubernetes native Jenkins Operator" \
      org.opencontainers.image.url="quay.io/jenkins-kubernetes-operator/operator" \
      org.opencontainers.image.source="https://github.com/jenkinsci/kubernetes-operator/tree/master" \
      org.opencontainers.image.base.name="gcr.io/distroless/static:nonroot"
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
