include variables.mk

.PHONY: all
all: status checkmake clean build verify install container-runtime-build container-runtime-images ## Build the image
	@echo "+ $@"

.PHONY: check-env
check-env: ## Checks the environment variables
	@echo "+ $@"
	@echo "NAME: $(NAME)"
ifeq ($(NAME),)
	$(error You must provide application name)
endif
	@echo "VERSION: $(VERSION)"
ifeq ($(VERSION),)
	$(error You must provide application version)
endif
	@echo "PKG: $(PKG)"
ifeq ($(PKG),)
	$(error You must provide application package)
endif
	@echo "VERSION_TAG: $(VERSION_TAG)"
	@echo "LATEST_TAG: $(LATEST_TAG)"
	@echo "BUILD_TAG: $(BUILD_TAG)"
ifneq ($(GITUNTRACKEDCHANGES),)
	@echo "Changes: \n$(GITUNTRACKEDCHANGES)"
endif

.PHONY: go-init
HAS_GIT := $(shell which git)
HAS_GO := $(shell which go)
go-init: ## Ensure build time dependencies
	@echo "+ $@"
ifndef HAS_GIT
	$(warning You must install git)
endif
ifndef HAS_GO
	$(warning You must install go)
endif

.PHONY: go-dependencies
go-dependencies: ## Ensure build dependencies
	@echo "+ $@"
	@echo "Ensure Golang runtime dependencies"
	go mod vendor -v

.PHONY: build
build: deepcopy-gen $(NAME) ## Builds a dynamic executable or package
	@echo "+ $@"

.PHONY: $(NAME)
$(NAME): $(wildcard *.go) $(wildcard */*.go) VERSION.txt
	@echo "+ $@"
	CGO_ENABLED=0 go build -tags "$(BUILDTAGS)" ${GO_LDFLAGS} -o bin/manager $(BUILD_PATH)

.PHONY: static
static: ## Builds a static executable
	@echo "+ $@"
	CGO_ENABLED=0 go build \
				-tags "$(BUILDTAGS) static_build" \
				${GO_LDFLAGS_STATIC} -o $(NAME) $(BUILD_PATH)

.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed
	@echo "+ $@"
	@go fmt $(PACKAGES)

.PHONY: lint
HAS_GOLINT := $(shell which $(PROJECT_DIR)/bin/golangci-lint)
lint: ## Verifies `golint` passes
	@echo "+ $@"
ifndef HAS_GOLINT
	$(call go-get-tool,$(PROJECT_DIR)/bin/golangci-lint,github.com/golangci/golangci-lint/cmd/golangci-lint@v1.26.0)
endif
	@bin/golangci-lint run

.PHONY: goimports
HAS_GOIMPORTS := $(shell which $(PROJECT_DIR)/bin/goimports)
goimports: ## Verifies `goimports` passes
	@echo "+ $@"
ifndef HAS_GOIMPORTS
	$(call go-get-tool,$(PROJECT_DIR)/bin/goimports,golang.org/x/tools/cmd/goimports@v0.1.0)
endif
	@bin/goimports -l -e $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: test
test: ## Runs the go tests
	@echo "+ $@"
	@RUNNING_TESTS=1 go test -tags "$(BUILDTAGS) cgo" $(PACKAGES_FOR_UNIT_TESTS)

.PHONY: e2e
e2e: deepcopy-gen manifests backup-kind-load ## Runs e2e tests, you can use EXTRA_ARGS
	@echo "+ $@"
	RUNNING_TESTS=1 go test -parallel=1 "./test/e2e/" -ginkgo.v -tags "$(BUILDTAGS) cgo" -v -timeout 60m -run "$(E2E_TEST_SELECTOR)" \
		-jenkins-api-hostname=$(JENKINS_API_HOSTNAME) -jenkins-api-port=$(JENKINS_API_PORT) -jenkins-api-use-nodeport=$(JENKINS_API_USE_NODEPORT) $(E2E_TEST_ARGS)

## Backup Section

.PHONY: backup-kind-load
backup-kind-load: ## Load latest backup image in the cluster
	@echo "+ $@"
	make -C backup/pvc backup-kind-load

## HELM Section

.PHONY: helm
HAS_HELM := $(shell command -v helm 2> /dev/null)
helm: ## Download helm if it's not present, otherwise symlink
	@echo "+ $@"
ifeq ($(strip $(HAS_HELM)),)
    mkdir -p $(PROJECT_DIR)/bin
    curl -Lo $(PROJECT_DIR)/bin/helm.tar.gz https://get.helm.sh/helm-v$(HELM_VERSION)-$(PLATFORM)-amd64.tar.gz && tar xzfv $(PROJECT_DIR)/bin/helm.tar.gz -C $(PROJECT_DIR)/bin
    mv $(PROJECT_DIR)/bin/$(PLATFORM)-amd64/helm $(PROJECT_DIR)/bin/helm
    rm -rf $(PROJECT_DIR)/bin/$(PLATFORM)-amd64
    rm -rf $(PROJECT_DIR)/bin/helm.tar.gz
else
	mkdir -p $(PROJECT_DIR)/bin
	test -L $(PROJECT_DIR)/bin/helm || ln -sf $(shell command -v helm) $(PROJECT_DIR)/bin/helm
endif

.PHONY: helm-lint
helm-lint: helm
	bin/helm lint chart/jenkins-operator

.PHONY: helm-release-latest
helm-release-latest: helm
	mkdir -p /tmp/jenkins-operator-charts
	mv chart/jenkins-operator/*.tgz /tmp/jenkins-operator-charts
	cd chart && ../bin/helm package jenkins-operator
	mv chart/jenkins-operator-*.tgz chart/jenkins-operator/
	bin/helm repo index chart/ --url https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart/ --merge chart/index.yaml
	mv /tmp/jenkins-operator-charts/*.tgz chart/jenkins-operator/

.PHONY: helm-e2e
IMAGE_NAME := quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY):$(GITCOMMIT)-amd64

helm-e2e: helm container-runtime-build-amd64 backup-kind-load ## Runs helm e2e tests, you can use EXTRA_ARGS
	kind load docker-image ${IMAGE_NAME} --name $(KIND_CLUSTER_NAME)
	@echo "+ $@"
	RUNNING_TESTS=1 go test -parallel=1 "./test/helm/" -ginkgo.v -tags "$(BUILDTAGS) cgo" -v -timeout 60m -run "$(E2E_TEST_SELECTOR)" -image-name=$(IMAGE_NAME) $(E2E_TEST_ARGS)

## CODE CHECKS section

.PHONY: vet
vet: ## Verifies `go vet` passes
	@echo "+ $@"
	@go vet $(PACKAGES)

.PHONY: staticcheck
HAS_STATICCHECK := $(shell which $(PROJECT_DIR)/bin/staticcheck)
staticcheck: ## Verifies `staticcheck` passes
	@echo "+ $@"
ifndef HAS_STATICCHECK
	$(eval TMP_DIR := $(shell mktemp -d))
	wget -O $(TMP_DIR)/staticcheck_$(PLATFORM)_amd64.tar.gz https://github.com/dominikh/go-tools/releases/download/2020.1.3/staticcheck_$(PLATFORM)_amd64.tar.gz
	tar zxvf $(TMP_DIR)/staticcheck_$(PLATFORM)_amd64.tar.gz -C $(TMP_DIR)
	mkdir -p $(PROJECT_DIR)/bin
	mv $(TMP_DIR)/staticcheck/staticcheck $(PROJECT_DIR)/bin
	rm -rf $(TMP_DIR)
endif
	@$(PROJECT_DIR)/bin/staticcheck $(PACKAGES)

.PHONY: cover
cover: ## Runs go test with coverage
	@echo "" > coverage.txt
	@for d in $(PACKAGES); do \
		IMG_RUNNING_TESTS=1 go test -race -coverprofile=profile.out -covermode=atomic "$$d"; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi; \
	done;

.PHONY: verify
verify: fmt lint test staticcheck vet ## Verify the code
	@echo "+ $@"

.PHONY: install
install: ## Installs the executable
	@echo "+ $@"
	go install -tags "$(BUILDTAGS)" ${GO_LDFLAGS} $(BUILD_PATH)

.PHONY: update-lts-version
update-lts-version: ## Update the latest lts version
	@echo "+ $@"
	echo $(LATEST_LTS_VERSION)
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' chart/jenkins-operator/values.yaml
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' test/e2e/test_utility.go
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' test/helm/helm_test.go
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' pkg/constants/constants.go
	#TODO: source the version from config.base.env for bats test, no need of hardcoded version
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' test/bats/1-deploy.bats
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' test/bats/2-deploy-with-more-options.bats
	sed -i 's|jenkins/jenkins:[0-9]\+.[0-9]\+.[0-9]\+|jenkins/jenkins:$(LATEST_LTS_VERSION)|g' test/bats/3-deploy-with-webhook.bats

.PHONY: run
run: export WATCH_NAMESPACE = $(NAMESPACE)
run: export OPERATOR_NAME = $(NAME)
run: fmt vet install-crds build ## Run the executable, you can use EXTRA_ARGS
	@echo "+ $@"
ifeq ($(KUBERNETES_PROVIDER),kind)
	kubectl config use-context $(KUBECTL_CONTEXT)
endif
ifeq ($(KUBERNETES_PROVIDER),crc)
	oc project $(CRC_OC_PROJECT)
endif
	@echo "Watching '$(WATCH_NAMESPACE)' namespace"
	bin/manager $(OPERATOR_ARGS)

.PHONY: clean
clean: ## Cleanup any build binaries or packages
	@echo "+ $@"
	go clean
	rm $(NAME) || echo "Couldn't delete, not there."
	rm -r $(BUILDDIR) || echo "Couldn't delete, not there."

.PHONY: spring-clean
spring-clean: ## Cleanup git ignored files (interactive)
	git clean -Xdi

define buildpretty
mkdir -p $(BUILDDIR)/$(1)/$(2);
GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 go build \
		-o $(BUILDDIR)/$(1)/$(2)/$(NAME) \
		-a -tags "$(BUILDTAGS) static_build netgo" \
		-installsuffix netgo ${GO_LDFLAGS_STATIC} $(BUILD_PATH);
md5sum $(BUILDDIR)/$(1)/$(2)/$(NAME) > $(BUILDDIR)/$(1)/$(2)/$(NAME).md5;
sha256sum $(BUILDDIR)/$(1)/$(2)/$(NAME) > $(BUILDDIR)/$(1)/$(2)/$(NAME).sha256;
endef

.PHONY: cross
cross: $(wildcard *.go) $(wildcard */*.go) VERSION.txt ## Builds the cross-compiled binaries, creating a clean directory structure (eg. GOOS/GOARCH/binary)
	@echo "+ $@"
	$(foreach GOOSARCH,$(GOOSARCHES), $(call buildpretty,$(subst /,,$(dir $(GOOSARCH))),$(notdir $(GOOSARCH))))

define buildrelease
GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 go build \
	 -o $(BUILDDIR)/$(NAME)-$(1)-$(2) \
	 -a -tags "$(BUILDTAGS) static_build netgo" \
	 -installsuffix netgo ${GO_LDFLAGS_STATIC} $(BUILD_PATH);
md5sum $(BUILDDIR)/$(NAME)-$(1)-$(2) > $(BUILDDIR)/$(NAME)-$(1)-$(2).md5;
sha256sum $(BUILDDIR)/$(NAME)-$(1)-$(2) > $(BUILDDIR)/$(NAME)-$(1)-$(2).sha256;
endef

.PHONY: release
release: $(wildcard *.go) $(wildcard */*.go) VERSION.txt ## Builds the cross-compiled binaries, naming them in such a way for release (eg. binary-GOOS-GOARCH)
	@echo "+ $@"
	$(foreach GOOSARCH,$(GOOSARCHES), $(call buildrelease,$(subst /,,$(dir $(GOOSARCH))),$(notdir $(GOOSARCH))))

.PHONY: checkmake
HAS_CHECKMAKE := $(shell which checkmake)
checkmake: ## Check this Makefile
	@echo "+ $@"
ifndef HAS_CHECKMAKE
	go get -u github.com/mrtazz/checkmake
endif
	@checkmake Makefile

.PHONY: container-runtime-login
container-runtime-login: ## Log in into the Docker repository
	@echo "+ $@"

.PHONY: container-runtime-build-%
container-runtime-build-%: ## Build the container
	@echo "+ $@"
	$(CONTAINER_RUNTIME_COMMAND) buildx build \
		--output=type=docker --platform linux/$* \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg CTIMEVAR="$(CTIMEVAR)" \
		--tag quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY):$(GITCOMMIT)-$* . \
		--file Dockerfile $(CONTAINER_RUNTIME_EXTRA_ARGS)

.PHONY: container-runtime-build
container-runtime-build: check-env deepcopy-gen container-runtime-build-amd64 container-runtime-build-arm64

.PHONY: container-runtime-images
container-runtime-images: ## List all local containers
	@echo "+ $@"
	$(CONTAINER_RUNTIME_COMMAND) images $(CONTAINER_RUNTIME_EXTRA_ARGS)

define buildx-create-command
$(CONTAINER_RUNTIME_COMMAND) buildx create \
	--driver=docker-container \
	--use
endef

## Parameter is version
define container-runtime-push-command
$(CONTAINER_RUNTIME_COMMAND) buildx build \
	--output=type=registry --platform linux/amd64,linux/arm64 \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg CTIMEVAR="$(CTIMEVAR)" \
	--tag quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY):$(1) . \
	--file Dockerfile $(CONTAINER_RUNTIME_EXTRA_ARGS)
endef

.PHONY: container-runtime-push
container-runtime-push: check-env deepcopy-gen ## Push the container
	@echo "+ $@"
	$(call buildx-create-command)
	$(call container-runtime-push-command,$(BUILD_TAG))

.PHONY: container-runtime-snapshot-push
container-runtime-snapshot-push: check-env deepcopy-gen
	@echo "+ $@"
	$(call buildx-create-command)
	$(call container-runtime-push-command,$(GITCOMMIT))

.PHONY: container-runtime-release-version
container-runtime-release-version: check-env deepcopy-gen ## Release image with version tag (in addition to build tag)
	@echo "+ $@"
	$(call buildx-create-command)
	$(call container-runtime-push-command,$(VERSION_TAG))

.PHONY: container-runtime-release-latest
container-runtime-release-latest: check-env deepcopy-gen ## Release image with latest tags (in addition to build tag)
	@echo "+ $@"
	$(call buildx-create-command)
	$(call container-runtime-push-command,$(LATEST_TAG))

.PHONY: container-runtime-release
container-runtime-release: container-runtime-release-version container-runtime-release-latest ## Release image with version and latest tags (in addition to build tag)
	@echo "+ $@"

# if this session isn't interactive, then we don't want to allocate a
# TTY, which would fail, but if it is interactive, we do want to attach
# so that the user can send e.g. ^C through.
INTERACTIVE := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(INTERACTIVE), 1)
    DOCKER_FLAGS += -t
endif

.PHONY: container-runtime-run
container-runtime-run: ## Run the container in docker, you can use EXTRA_ARGS
	@echo "+ $@"
	$(CONTAINER_RUNTIME_COMMAND) run $(CONTAINER_RUNTIME_EXTRA_ARGS) --rm -i $(DOCKER_FLAGS) \
		--volume $(HOME)/.kube/config:/home/jenkins-operator/.kube/config \
		quay.io/${QUAY_ORGANIZATION}/$(QUAY_REGISTRY):$(GITCOMMIT) /usr/bin/jenkins-operator $(OPERATOR_ARGS)

.PHONY: crc-run
crc-run: export WATCH_NAMESPACE = $(NAMESPACE)
crc-run: export OPERATOR_NAME = $(NAME)
crc-run: crc-start run ## Run the operator locally and use CodeReady Containers as Kubernetes cluster, you can use OPERATOR_ARGS
	@echo "+ $@"

.PHONY: deepcopy-gen
deepcopy-gen: generate ## Generate deepcopy golang code
	@echo "+ $@"

.PHONY: scheme-doc-gen
HAS_GEN_CRD_API_REFERENCE_DOCS := $(shell ls gen-crd-api-reference-docs 2> /dev/null)
scheme-doc-gen: ## Generate Jenkins CRD scheme doc
	@echo "+ $@"
ifndef HAS_GEN_CRD_API_REFERENCE_DOCS
	@wget https://github.com/ahmetb/$(GEN_CRD_API)/releases/download/v0.1.2/$(GEN_CRD_API)_$(PLATFORM)_amd64.tar.gz
	@mkdir -p $(GEN_CRD_API)
	@tar -C $(GEN_CRD_API) -zxf $(GEN_CRD_API)_$(PLATFORM)_amd64.tar.gz
	@rm $(GEN_CRD_API)_$(PLATFORM)_amd64.tar.gz
endif
	$(GEN_CRD_API)/$(GEN_CRD_API) -config gen-crd-api-config.json -api-dir $(PKG)/api/$(API_VERSION) -template-dir $(GEN_CRD_API)/template -out-file documentation/$(VERSION)/jenkins-$(API_VERSION)-scheme.md

.PHONY: check-crc
check-crc: ## Checks if KUBERNETES_PROVIDER is set to crc
	@echo "+ $@"
	@echo "KUBERNETES_PROVIDER '$(KUBERNETES_PROVIDER)'"
ifneq ($(KUBERNETES_PROVIDER),crc)
	$(error KUBERNETES_PROVIDER not set to 'crc')
endif

.PHONY: kind-setup
kind-setup: ## Setup kind cluster
	@echo "+ $@"
	kind create cluster --config kind-cluster.yaml --name $(KIND_CLUSTER_NAME)

.PHONY: kind-clean
kind-clean: ## Delete kind cluster
	@echo "+ $@"
	kind delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: bats-tests
IMAGE_NAME := quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY):$(GITCOMMIT)-amd64
BUILD_PRESENT := $(shell docker images |grep -q ${IMAGE_NAME})
ifndef BUILD_PRESENT
bats-tests: backup-kind-load container-runtime-build-amd64 ## Run bats tests
	@echo "+ $@"
	kind load docker-image ${IMAGE_NAME} --name $(KIND_CLUSTER_NAME)
	OPERATOR_IMAGE="${IMAGE_NAME}" TERM=xterm bats -T -p test/bats
else
bats-tests: backup-kind-load
	@echo "+ $@"
	OPERATOR_IMAGE="${IMAGE_NAME}" TERM=xterm bats -T -p test/bats
endif

.PHONY: crc-start
crc-start: check-crc ## Start CodeReady Containers Kubernetes cluster
	@echo "+ $@"
	crc start

.PHONY: sembump
HAS_SEMBUMP := $(shell which $(PROJECT_DIR)/bin/sembump)
sembump: # Download sembump locally if necessary
	@echo "+ $@"
ifndef HAS_SEMBUMP
	mkdir -p $(PROJECT_DIR)/bin
	wget -O $(PROJECT_DIR)/bin/sembump https://github.com/justintout/sembump/releases/download/v0.1.0/sembump-$(PLATFORM)-amd64
	chmod +x $(PROJECT_DIR)/bin/sembump
endif

.PHONY: bump-version
BUMP := patch
bump-version: sembump ## Bump the version in the version file. Set BUMP to [ patch | major | minor ]
	@echo "+ $@"
	$(eval NEW_VERSION=$(shell bin/sembump --kind $(BUMP) $(VERSION)))
	@echo "Bumping VERSION.txt from $(VERSION) to $(NEW_VERSION)"
	echo $(NEW_VERSION) > VERSION.txt
	@echo "Updating version from $(VERSION) to $(NEW_VERSION) in README.md"
	sed -i.bak 's/$(VERSION)/$(NEW_VERSION)/g' README.md
	sed -i.bak 's/$(VERSION)/$(NEW_VERSION)/g' config/manager/manager.yaml
	sed -i.bak 's/$(VERSION)/$(NEW_VERSION)/g' deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	rm */*/**.bak
	rm */**.bak
	rm *.bak
	cp config/service_account.yaml deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	cat config/rbac/leader_election_role.yaml >> deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	cat config/rbac/leader_election_role_binding.yaml >> deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	cat config/rbac/role.yaml >> deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	cat config/rbac/role_binding.yaml >> deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	cat config/manager/manager.yaml >> deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	git add VERSION.txt README.md config/manager/manager.yaml deploy/$(ALL_IN_ONE_DEPLOY_FILE_PREFIX)-$(API_VERSION).yaml
	git commit -vaem "Bump version to $(NEW_VERSION)"
	@echo "Run make tag to create and push the tag for new version $(NEW_VERSION)"

.PHONY: tag
tag: ## Create a new git tag to prepare to build a release
	@echo "+ $@"
	git tag -a $(VERSION) -m "$(VERSION)"
	git push origin $(VERSION)

.PHONY: help
help:
	@grep -Eh '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: status
status: ## Shows git status
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
	go mod vendor -v
	@echo


# Download and build hugo extended locally if necessary
HUGO_PATH = $(shell pwd)/bin/hugo
HUGO_VERSION = v0.99.1
HAS_HUGO := $(shell $(HUGO_PATH)/hugo version 2>&- | grep $(HUGO_VERSION))
hugo:
ifeq ($(HAS_HUGO), )
	@echo "Installing Hugo $(HUGO_VERSION)"
	rm -rf $(HUGO_PATH)
	git clone https://github.com/gohugoio/hugo.git --depth=1 --branch $(HUGO_VERSION) $(HUGO_PATH)
	cd $(HUGO_PATH) && go build --tags extended -o hugo main.go
endif

.PHONY: generate-docs
generate-docs: hugo ## Re-generate docs directory from the website directory
	@echo "+ $@"
	rm -rf docs || echo "Cannot remove docs dir, ignoring"
	cd website && npm install
	$(HUGO_PATH)/hugo -s website -d ../docs

.PHONY: run-docs
run-docs: hugo
	@echo "+ $@"
	cd website && $(HUGO_PATH)/hugo server -D

##################### FROM OPERATOR SDK ########################
# Install CRDs into a cluster
install-crds: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall-crds: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY):$(GITCOMMIT)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: operator-sdk
HAS_OPERATOR_SDK := $(shell which $(PROJECT_DIR)/bin/operator-sdk)
operator-sdk: # Download operator-sdk locally if necessary
	@echo "+ $@"
ifndef HAS_OPERATOR_SDK
	wget -O $(PROJECT_DIR)/bin/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v1.3.0/operator-sdk_$(PLATFORM)_amd64
	chmod +x $(PROJECT_DIR)/bin/operator-sdk
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests operator-sdk kustomize
	bin/operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/$(QUAY_ORGANIZATION)/$(QUAY_REGISTRY):$(VERSION_TAG)
	$(KUSTOMIZE) build config/manifests | bin/operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	bin/operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Download kubebuilder
kubebuilder:
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR);

# install cert-manager v1.5.1
install-cert-manager: kind-setup
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.5.1/cert-manager.yaml

uninstall-cert-manager: kind-setup
	kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v1.5.1/cert-manager.yaml

# Deploy the operator locally along with webhook using helm charts
deploy-webhook: container-runtime-build-amd64
	@echo "+ $@"
	bin/helm upgrade jenkins chart/jenkins-operator --install --set-string operator.image=${IMAGE_NAME} --set webhook.enabled=true --set jenkins.enabled=false
