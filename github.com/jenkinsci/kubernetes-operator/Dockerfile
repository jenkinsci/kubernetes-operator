FROM docker:18.09

ARG GO_VERSION
ARG OPERATOR_SDK_VERSION
ARG MINIKUBE_VERSION

ARG GOPATH="/go"

RUN mkdir -p /go

# Stage 1 - Install dependencies
RUN apk update && \
    apk add --no-cache \
            curl \
            python \
            py-crcmod \
            bash \
            libc6-compat \
            openssh-client \
            git \
            make \
            gcc \
            libc-dev \
            git

RUN curl -O https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz && tar -xvf go$GO_VERSION.linux-amd64.tar.gz

# Stage 2 - Install operator-sdk
RUN echo $GOPATH/bin/operator-sdk
RUN curl -L https://github.com/operator-framework/operator-sdk/releases/download/v$OPERATOR_SDK_VERSION/operator-sdk-v$OPERATOR_SDK_VERSION-x86_64-linux-gnu -o $GOPATH/bin/operator-sdk \
    && chmod +x $GOPATH/bin/operator-sdk

RUN curl -Lo minikube https://storage.googleapis.com/minikube/releases/v$MINIKUBE_VERSION/minikube-linux-amd64 \
    && chmod +x minikube \
    && cp minikube /usr/local/bin/ \
    && rm minikube

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

RUN export GO111MODULE=auto

RUN mkdir -p $GOPATH/src/github.com/jenkinsci/kubernetes-operator
ADD . $GOPATH/src/github.com/jenkinsci/kubernetes-operator
WORKDIR $GOPATH/src/github.com/jenkinsci/kubernetes-operator

RUN mkdir -p /home/builder

ENV DOCKER_TLS_VERIFY   1
ENV DOCKER_CERT_PATH    /minikube/certs

ENTRYPOINT ["./entrypoint.sh"]
