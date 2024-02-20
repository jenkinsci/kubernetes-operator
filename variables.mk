SHELL := /usr/bin/env bash
PATH  := $(GOPATH)/bin:$(PATH)

OSFLAG 				:=
ifeq ($(OS),Windows_NT)
	OSFLAG = WIN32
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		OSFLAG = LINUX
	endif
	ifeq ($(UNAME_S),Darwin)
		OSFLAG = OSX
	endif
endif

define strip_quotes
$(shell echo $(1) | sed -e 's/^"//' -e 's/"$$//')
endef

include config.base.env
$(foreach var,$(shell cat config.base.env),$(eval $(call strip_quotes,$(var))))

# Import config
# You can change the default config with `make config="config_special.env" build`
config ?= config.minikube.env
include $(config)

# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

VERSION := $(shell cat VERSION.txt)
GITCOMMIT := $(shell git rev-parse --short HEAD)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
GITIGNOREDBUTTRACKEDCHANGES := $(shell git ls-files -i -c --exclude-standard)
ifneq ($(GITUNTRACKEDCHANGES),)
    GITCOMMIT := $(GITCOMMIT)-dirty
endif
ifneq ($(GITIGNOREDBUTTRACKEDCHANGES),)
    GITCOMMIT := $(GITCOMMIT)-dirty
endif

VERSION_TAG := $(VERSION)
LATEST_TAG := latest
BUILD_TAG := $(GITBRANCH)-$(GITCOMMIT)

BUILD_PATH := ./

# CONTAINER_RUNTIME_COMMAND is Container Runtime - it could be docker or podman
CONTAINER_RUNTIME_COMMAND := docker

# Set any default go build tags
BUILDTAGS :=

# Set the build dir, where built cross-compiled binaries will be output
BUILDDIR := ${PREFIX}/cross

CTIMEVAR=-X $(PKG)/version.GitCommit=$(GITCOMMIT) -X $(PKG)/version.Version=$(VERSION)
GO_LDFLAGS=-ldflags "-w $(CTIMEVAR)"
GO_LDFLAGS_STATIC=-ldflags "-w $(CTIMEVAR) -extldflags -static"

# List the GOOS and GOARCH to build
GOOSARCHES = linux/amd64

PACKAGES = $(shell go list -f '{{.ImportPath}}/' ./... | grep -v vendor)
PACKAGES_FOR_UNIT_TESTS = $(shell go list -f '{{.ImportPath}}/' ./... | grep -v vendor | grep -v e2e | grep -v helm)

# Run all the e2e tests by default
E2E_TEST_SELECTOR ?= .*

JENKINS_API_HOSTNAME := $(shell $(JENKINS_API_HOSTNAME_COMMAND) 2> /dev/null || echo "" )
OPERATOR_ARGS ?= --jenkins-api-hostname=$(JENKINS_API_HOSTNAME) --jenkins-api-port=$(JENKINS_API_PORT) --jenkins-api-use-nodeport=$(JENKINS_API_USE_NODEPORT) --cluster-domain=$(CLUSTER_DOMAIN) $(OPERATOR_EXTRA_ARGS)

.DEFAULT_GOAL := help

PLATFORM = $(shell echo $(UNAME_S) | tr A-Z a-z)
CPUS_NUMBER = 3
MEMORY_AMOUNT = 4096
##################### FROM OPERATOR SDK ########################

# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
