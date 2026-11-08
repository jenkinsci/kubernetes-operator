# Contribution Model

Thanks for taking the time to contribute!

## Code of Conduct

This project and everyone participating in it is governed by the [Jenkins Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## We Develop with GitHub
We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## We Use GitHub Flow, So All Code Changes Happen Through Pull Requests
Pull requests are the best way to propose changes to the codebase (we use GitHub Flow). We actively welcome your pull requests:

- Create feature requests and discuss the scope.
- Fork the repo and create your branch from master.
- If you’ve added code that should be tested, add tests.
- If you’ve changed APIs or design, update the documentation.
- Create a draft pull request (not yet ready for review, triggers CI build).
- Ensure the e2e tests pass (wait for GitHub status checks).
- Mark that pull request as ready for review.

## Development setup

Choose the environment that suits you:

- **Nix devshell:** run `nix develop` from the repository root for Go, Kind,
  pre-commit, helm-docs, Bats and the other development tools. Provide a running
  Docker engine and client, Git and kubectl on the host.
- **Devcontainer:** open the checkout in VS Code and select **Dev Containers:
  Reopen in Container**, using `.devcontainer/devcontainer.json`. It configures
  Go, Docker-in-Docker, kubectl, Helm, Kind, Bats and Nix. Run `nix develop`
  in its terminal when you need the full development toolset, including pre-commit
  and helm-docs.
- **Local tools with Make:** install Git, GNU Make, Bash, Go, Docker, kubectl and
  Kind. Make installs versioned build tools such as controller-gen, kustomize,
  golangci-lint and staticcheck as their targets need them. Install pre-commit
  and helm-docs separately for the commit hooks. Make does not install every
  prerequisite.

Use `config.base.env` and `variables.mk` for the configured tool versions. The
[Developer Guide](https://jenkinsci.github.io/kubernetes-operator/docs/developer-guide/)
covers setup, cluster tests and backup tests for these options. Website development
and publishing use the dedicated Nix website environment.

Run focused tests first. For Go changes, run `make verify`; for API changes, also
run `make manifests` and review the generated files. Run local integration suites
sequentially in one Kind cluster; CI keeps its scenarios isolated.

Install and run the hooks before submitting:

```bash
pre-commit install
```

For chart changes, run `make helm-lint` and regenerate its README with
`helm-docs --chart-search-root=chart/jenkins-operator`. The Helm hook requires
`helm` on PATH; with Make-installed Helm, use `export PATH="$PWD/bin:$PATH"`.
Do not commit local plugin-mirror overrides or unrelated generated files.

## Commit messages

Start with a concise subject explaining the change and reference the relevant
issue. Use `Fixes #123` only when the change resolves the entire issue; otherwise
use `Refs #123` and describe the remaining work in the PR.

## Quality Standards
It is important to keep the quality bar high and ensure all of us follow best practices and security, so the code is solid and can be reused by other people. Below you can find some quality standards.

### General Contribution

- Break down your pull request into smaller pieces, less code is easier to review. Most of PRs should fall under 200 lines of code changes unless specifically justified otherwise.
- Add descriptive comments and labels to pull requests and issues.
- Ensure end to end tests are passing.
- commit message should follow [these best practices](https://chris.beams.io/posts/git-commit/), specifically: start with a subject line and reference an issue number e.g. ["Fixes #245" or "Closes #111"](https://help.github.com/articles/closing-issues-using-keywords/)

### Large Architectural Changes
In case of large change which requires significant engineering effort and introduces side effects, we suggest writing a design proposal document first.

Design proposal is simply a document that states what you propose to do including:

- Problem statement
- Description of potential solution
- Side effects
- Breaking changes

Keep in mind that a proposal is not a pitch, it needs to be technical.

### Proposing changes to Helm Chart
When issuing a PR that modifies the project's Helm Chart, please do not include in your PR changes that would release a new package version when merged.

Specifically, please do not update `chart/index.yaml` and `chart/jenkins-operator/Chart.yaml` files and do not build chart archive package.

For the sake of PR's brevity and security, Project's maintainers will issue a separate PR that releases new version of the Chart after your PR has been merged.
