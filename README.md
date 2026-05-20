# Podplane Workspace

Tooling for managing all Podplane-related Git repositories across:

* [podplane](https://github.com/orgs/podplane/repositories)
* [netsy-dev](https://github.com/orgs/netsy-dev/repositories)
* [nstance-dev](https://github.com/orgs/nstance-dev/repositories)
* [easy-oidc](https://github.com/orgs/easy-oidc/repositories)
* [puidv7](https://github.com/orgs/puidv7/repositories)

Run `make help` to see available commands.

## GitHub Audit

Run `make github-audit` to check every managed org and repository against
[`github-policy.jsonc`](github-policy.jsonc). The audit is read-only and uses the
official `gh` CLI, so you should run `gh auth login` first.

The audit checks organization-level GitHub App requirements first, then prints a
grouped result for each repository as soon as that repository finishes. It covers
repository rulesets, DCO status-check requirements, release immutability, common
repository feature toggles, repository visibility, pull request creation and
merge settings, and web commit signoff.
