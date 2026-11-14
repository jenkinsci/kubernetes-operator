---
title: "Developer Guide"
linkTitle: "Developer Guide"
weight: 5
date: 2026-10-31
description: >
  Build and test Jenkins Operator with Nix, a devcontainer, or local tools
---

## Development environment

Choose the Nix devshell, the repository devcontainer, or local tools with Make.
All three use the same build and test targets below.

### Nix devshell

Install Git, Nix with flakes enabled, a running Docker engine and client, and `kubectl`.
Clone the repository and enter the development shell:

```bash
git clone git@github.com:jenkinsci/kubernetes-operator.git
cd kubernetes-operator
nix develop
```

The default shell provides Go, Kind, pre-commit, helm-docs, Bats and its libraries, and other
build tools. Tool versions are maintained in
[`config.base.env`](https://github.com/jenkinsci/kubernetes-operator/blob/v0.10.0/config.base.env)
and consumed by `variables.mk`; do not copy separate version settings into local scripts.
Release 0.10.0 uses Go 1.26 and Operator SDK 1.42.3.

### Devcontainer

Open the checkout in VS Code with the Dev Containers extension and select
**Dev Containers: Reopen in Container**. The configuration in `.devcontainer/devcontainer.json`
provides Go, Docker-in-Docker, kubectl, Helm, Kind, Bats and its libraries, and Nix.
Run `nix develop` in the container terminal for the full development toolset,
including pre-commit and helm-docs. Check `docker info` before running cluster tests.

### Local tools with Make

Nix is not required by CI or the Dockerfiles. For a local setup, install the Go version from `config.base.env`, Git, GNU Make, Bash, Docker, `kubectl`,
and Kind. The Make targets download versioned tools such as Helm, controller-gen, kustomize,
golangci-lint and staticcheck into `bin/` as needed. Make does not install the host
prerequisites listed above.
Install helm-docs and pre-commit as well if you use the documentation and commit hooks.

## Build and run with Kind

Create one local cluster and build the operator:

```bash
make kind-setup
make build
```

The default cluster is `jenkins`, with context `kind-jenkins`. Confirm that this is the context
you intend to use, then run the operator locally:

```bash
make run
```

`make run` installs CRDs, builds the operator, and watches the namespace selected by `NAMESPACE`
(`default` unless overridden). In another terminal, apply a Jenkins resource from the
[deployment guide](/kubernetes-operator/docs/getting-started/latest/deploying-jenkins/).
The operator must remain running while it configures Jenkins.

For an existing cluster, select its context before running Make so CRD installation uses the
same cluster as the operator:

```bash
kubectl config use-context my-context
make run KUBECTL_CONTEXT=my-context NAMESPACE=jenkins
```

Create the namespace first if it does not exist. For Docker Desktop's Kubernetes cluster,
use `make config=config.docker-desktop.env run` after selecting its context.

## Focused tests and verification

Run a focused package test before the complete verification target:

```bash
RUNNING_TESTS=1 go test ./pkg/plugins
make verify
```

`make verify` runs formatting, lint, unit tests, staticcheck, and vet. The formatting target may
modify files; review the diff afterward. For unit tests alone, use `make test`.

Before committing, run the repository hooks against the files you changed:

```bash
pre-commit install
```

For chart changes, use `make helm-lint` and regenerate the chart README with
`helm-docs --chart-search-root=chart/jenkins-operator`. If using Make-installed Helm,
run `export PATH="$PWD/bin:$PATH"` before the hooks so the Helm hook can find it.

## Cluster integration tests

Use the same local Kind cluster and run suites sequentially. These tests create and remove
Kubernetes resources; use a development cluster rather than an installation containing Jenkins data.

```bash
make kind-setup  # only if the local cluster does not already exist
kubectl config use-context kind-jenkins
make e2e
make helm-e2e
make bats-tests
```

To focus on a Ginkgo example, pass a matching description through `E2E_TEST_ARGS`:

```bash
make e2e E2E_TEST_ARGS='-ginkgo.focus=creates.*vanilla'
```

`E2E_TEST_SELECTOR` selects the Go test entry point with `-run`; Ginkgo example selection uses
`-ginkgo.focus`. CI runs scenarios in separate jobs; do not run those scenarios concurrently
against your shared local cluster. Keep any local plugin-mirror overrides out of committed fixtures.

The Helm/Bats Make targets currently build AMD64 operator images. Use an AMD64 test environment
for those targets. The backup image has separate AMD64 and ARM64 checks.

## Backup image tests

`make -C backup/pvc docker-e2e` builds the local-platform image and runs its backup/restore tests.
The test image's user must be able to write the bind-mounted test directories; adjust its UID/GID
for your machine. These tests do not need a Kubernetes cluster.

To test an ARM64 image without pushing to a registry, use Docker with ARM64 execution support
(native or QEMU/binfmt) and load a single-platform image:

```bash
make -C backup/pvc docker-buildx PLATFORMS=linux/arm64 \
  BUILDX_OUTPUT=type=docker GITCOMMIT=backup-dev UID="$(id -u)" GID=1000
DOCKER_DEFAULT_PLATFORM=linux/arm64 make -C backup/pvc docker-test GITCOMMIT=backup-dev
```

Use an available group ID in the image; `1000` is the normal backup image default. For an AMD64
check, substitute `linux/amd64`. Run the two suites sequentially because they share fixture directories.
The `docker-test` target tests the loaded image without rebuilding it for the host architecture.

## API and generated files

After changing API types, regenerate the deepcopy code and manifests:

```bash
make generate
make manifests
make verify
```

Review the generated CRDs, RBAC, and chart/deploy copies with the source changes. Do not edit generated
files to introduce changes that their source cannot reproduce.

## Access Jenkins locally

For a Jenkins resource named `example` in `default`, forward the HTTP service:

```bash
kubectl --namespace default port-forward service/jenkins-operator-http-example 8080:8080
```

Open `http://localhost:8080`. In another terminal, read the generated credentials:

```bash
kubectl --namespace default get secret jenkins-operator-credentials-example -o 'jsonpath={.data.user}' | base64 -d
kubectl --namespace default get secret jenkins-operator-credentials-example -o 'jsonpath={.data.password}' | base64 -d
```

## Website development

From the repository root, enter the website directory before opening its shell:

```bash
cd website
nix develop ..#website
hugo server -D
```

This shell links the pinned Docsy source from the Nix store and installs the website's npm dependencies with `npm ci`. Review incidental package/lockfile changes
before committing. For a production build from the repository root, use `nix build .#website`.
New files must be added to Git for a Git-backed flake build to include them.

Website development and publishing require Nix. The dedicated website shell provides
Hugo, Node.js/npm and Dart Sass. The root Make targets are shortcuts for the Nix workflow:

```bash
make generate-docs  # builds the production website in result/
make run-docs       # starts the preview with drafts enabled
```

To update Docsy, change the `docsy` input in `flake.nix`, update its lock with
`nix flake update docsy`, and review its required Hugo and frontend dependency versions.
Keep `HUGO_VERSION` and `DART_SASS_VERSION` in `config.base.env` aligned with the
pinned Nix packages.
After npm dependency changes, refresh the website's `npmDepsHash` in
`nix/website-build.nix` and verify `nix build .#website`.

## Clean up the local cluster

Once all local cluster tests are finished:

```bash
make kind-clean
```
