# Set POSIX sh for maximum interoperability
SHELL := /bin/sh
PATH  := $(GOPATH)/bin:$(PATH)

# Import config
# You can change the default config with `make config="config_special.env" build`
config ?= config.env
include $(config)

# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

VERSION := $(shell cat VERSION.txt)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
GITCOMMIT := $(shell git rev-parse --short HEAD)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
GITIGNOREDBUTTRACKEDCHANGES := $(shell git ls-files -i -o --exclude-standard)
ifneq ($(GITUNTRACKEDCHANGES),)
    GITCOMMIT := $(GITCOMMIT)-dirty
endif
ifneq ($(GITIGNOREDBUTTRACKEDCHANGES),)
    GITCOMMIT := $(GITCOMMIT)-dirty
endif

VERSION_TAG := $(VERSION)
LATEST_TAG := latest
BUILD_TAG := $(GITBRANCH)-$(GITCOMMIT)

ARGS ?= $(EXTRA_ARGS)

.DEFAULT_GOAL := help

.PHONY: all
all: status checkmake clean build verify install docker-build docker-images ## Build the image
	@echo "+ $@"

.PHONY: check-env
check-env: ## Checks the environment variables
	@echo "+ $@"
	@echo "NAME: $(NAME)"
ifeq ($(NAME),)
	$(error You must provide name)
endif
	@echo "VERSION: $(VERSION)"
ifeq ($(VERSION),)
	$(error You must provide version)
endif
	@echo "VERSION_TAG: $(VERSION_TAG)"
	@echo "LATEST_TAG: $(LATEST_TAG)"
	@echo "BUILD_TAG: $(BUILD_TAG)"
ifneq ($(GITUNTRACKEDCHANGES),)
	@echo "Changes: \n$(GITUNTRACKEDCHANGES)"
endif

.PHONY: spring-clean
spring-clean: ## Cleanup git ignored files (interactive)
	git clean -Xdi

.PHONY: checkmake
HAS_CHECKMAKE := $(shell command -v checkmake)
checkmake: ## Check this Makefile
	@echo "+ $@"
ifndef HAS_CHECKMAKE
	go get -u github.com/mrtazz/checkmake
endif
	@checkmake Makefile

define e2e
echo "\nRunning $(1) e2e test";
@e2e/$(1)/test.sh quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT)
endef

.PHONY: docker-e2e
E2E_TESTS := $(shell ls e2e)
docker-e2e: docker-build-e2e ## Make e2e tests for docker image
	@echo "+ $@"
	$(foreach TEST_NAME,$(E2E_TESTS), $(call e2e,$(TEST_NAME)))

.PHONY: docker-login
docker-login: ## Log in into the Docker repository
	@echo "+ $@"

.PHONY: docker-build-e2e
docker-build-e2e: UID=1001
docker-build-e2e: GID=1001
docker-build-e2e: docker-build

.PHONY: docker-build
docker-build: check-env ## Build the container
	@echo "+ $@"
	docker build . --build-arg UID=$(UID) --build-arg GID=$(GID) -t quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT) --file Dockerfile

.PHONY: docker-images
docker-images: ## List all local containers
	@echo "+ $@"
	docker images

.PHONY: docker-push
docker-push: docker-build ## Push the container
	@echo "+ $@"
	docker tag quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT) quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(BUILD_TAG)
	docker push quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(BUILD_TAG)

.PHONY: docker-release-version
docker-release-version: docker-build ## Release image with version tag (in addition to build tag)
	@echo "+ $@"
	docker tag quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT) quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(VERSION_TAG)
	docker push quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(VERSION_TAG)

.PHONY: docker-release-latest
docker-release-latest: docker-build ## Release image with latest tags (in addition to build tag)
	@echo "+ $@"
	docker tag quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT) quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(LATEST_TAG)
	docker push quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(LATEST_TAG)

.PHONY: docker-release
docker-release: docker-release-version docker-release-latest ## Release image with version and latest tags (in addition to build tag)
	@echo "+ $@"

.PHONY: backup-kind-load
backup-kind-load: docker-build ## Build and load backup img in kind with e2e-test tag
	@echo "+ $@"
	docker tag quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT) quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):e2e-test
	kind load docker-image quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):e2e-test --name $(KIND_CLUSTER_NAME)

# if this session isn't interactive, then we don't want to allocate a
# TTY, which would fail, but if it is interactive, we do want to attach
# so that the user can send e.g. ^C through.
INTERACTIVE := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(INTERACTIVE), 1)
    DOCKER_FLAGS += -t
endif

.PHONY: docker-run
docker-run: docker-build
	@echo "+ $@"
	docker run --rm -i $(DOCKER_FLAGS) \
		quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY)-$(NAME):$(GITCOMMIT) $(ARGS)

.PHONY: sembump
HAS_SEMBUMP := $(shell which $(PROJECT_DIR)/bin/sembump)
sembump: # Download sembump locally if necessary
	@echo "+ $@"
ifndef HAS_SEMBUMP
	mkdir -p $(PROJECT_DIR)/bin
	wget -O $(PROJECT_DIR)/bin/sembump https://github.com/justintout/sembump/releases/download/v0.1.0/sembump-linux-amd64
	chmod +x $(PROJECT_DIR)/bin/sembump
endif

.PHONY: bump-version
BUMP := patch
bump-version: sembump ## Bump the version in the version file. Set BUMP to [ patch | major | minor ]
	@echo "+ $@"
	$(eval NEW_VERSION=$(shell $(PROJECT_DIR)/bin/sembump --kind $(BUMP) $(VERSION)))
	@echo "Bumping VERSION.txt from $(VERSION) to $(NEW_VERSION)"
	echo $(NEW_VERSION) > VERSION.txt
	git add VERSION.txt
	git commit -avm "Bump backup PVC version to $(NEW_VERSION)"

.PHONY: tag
tag: ## Create a new git tag to prepare to build a release
	@echo "+ $@"
	git tag -s -a $(VERSION) -m "$(VERSION)"
	git push origin $(VERSION)

.PHONY: help
help:
	@grep -Eh '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: status
status: ## Shows git and dep status
	@echo "+ $@"
	@echo "Commit: $(GITCOMMIT), VERSION: $(VERSION)"
	@echo
ifneq ($(GITUNTRACKEDCHANGES),)
	@echo "Changed files:"
	@git status --porcelain --untracked-files=no
	@echo
endif
ifneq ($(GITIGNOREDBUTTRACKEDCHANGES),)
	@echo "Ignored but tracked files:"
	@git ls-files -i -c --exclude-standard
	@echo
endif
	@echo "Dependencies:"
	dep status
	@echo
