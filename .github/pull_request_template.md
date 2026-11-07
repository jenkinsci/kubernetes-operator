<!-- 🎉🎉🎉 Thank you for the PR!!! 🎉🎉🎉 -->

# Changes

<!-- Describe your changes here- ideally you can get that description straight from
your descriptive commit message(s)! -->

# Validation

<!-- Describe the checks you ran and their results. State when a check was not run.
For development setup, see CONTRIBUTING.md: Nix, devcontainer, or local tools with Make. -->

# Submitter Checklist

These are the criteria that every PR should meet, please check them off as you
review them:

- [ ] Includes tests (if functionality changed/added)
- [ ] Includes docs (if user facing)
- [ ] Relevant pre-commit hooks pass
- [ ] `make verify` passes (if Go code changed)
- [ ] `make manifests` run and generated changes reviewed (if API changed)
- [ ] Chart lint and generated README updated (if chart changed)
- [ ] Commit messages follow [commit message best practices](https://github.com/jenkinsci/kubernetes-operator/blob/master/CONTRIBUTING.md#commit-messages)

_See [the contribution guide](https://github.com/jenkinsci/kubernetes-operator/blob/master/CONTRIBUTING.md) for more details._
