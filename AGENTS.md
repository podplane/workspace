# Podplane Workspace — Agent Guide

This repository contains tooling for coordinating work across the Podplane-related GitHub organizations and repositories. It is intentionally lightweight: its main job is to help developers and coding agents know which repositories exist, where they are checked out locally, and which project owns which responsibility.

## Workspace Layout

Assume all repositories are checked out under a common workspace directory:

```text
~/Workspace/<github-org>/<github-repo>
```

For example:

```text
~/Workspace/podplane/workspace      -> git@github.com:podplane/workspace.git
~/Workspace/podplane/podplane       -> git@github.com:podplane/podplane.git
~/Workspace/nstance-dev/nstance     -> git@github.com:nstance-dev/nstance.git
~/Workspace/netsy-dev/netsy         -> git@github.com:netsy-dev/netsy.git
```

Use this repo's `Makefile` as the source of truth for the current managed repository list.

## Commands

- `make list` — list managed repositories grouped by GitHub organization/project.
- `make status` — quickly report clone coverage, fetch remotes, then show branch/sync/change status for each repo.
- `make status FETCH=0` — same status report without fetching remotes.
- `make clone` — clone any missing repositories using SSH URLs.

## Cross-Project Orientation

When the user describes a cross-component task, map project names to repositories like this:

### Podplane

- `podplane/workspace` — this repository; workspace tooling and cross-repo coordination.
- `podplane/podplane` — main Podplane product repository. Treat mentions of "Podplane", "Podplane CLI", "the CLI", "Podplane docs", or core Podplane orchestration as likely referring here unless context says otherwise.
- `podplane/vmconfig` — Virtual Machine configuration system for Podplane. Treat mentions of "vmconfig", "VM config", "userdata", or configuring/running VM-backed workloads for Podplane as likely referring here.
- `podplane/components` — Helm charts and configuration for the in-cluster components (cert-manager, Cilium, CoreDNS, Traefik, Gateway API CRDs, AWS EBS CSI driver, etc.) that turn a bare Podplane Kubernetes cluster into a PaaS. Treat mentions of "components", "Helm charts", "Flux apps", "cluster add-ons", or specific bundled components (cert-manager, Cilium, CoreDNS, Traefik, trust-manager, snapshot controller, EBS CSI) in a Podplane context as likely referring here.
- `podplane/workers` — Cloudflare Workers monorepo for Podplane-related edge/background services.
- `podplane/website` — current Podplane website.

### Netsy

- `netsy-dev/netsy` — replicated key-value database backed by object storage; etcd-compatible persistence target for Kubernetes/Podplane scenarios. Treat mentions of "netsy" or "netsy docs" as likely referring here unless context says otherwise.
- `netsy-dev/website` — Netsy website.

### Nstance

- `nstance-dev/nstance` — VM autoscaling/orchestration system for AWS, Google Cloud, Proxmox, Kubernetes, and local dev providers. Treat mentions of "nstance-server", "nstance-agent", "nstance-operator", "nstance docs", VM lifecycle, autoscaling, or instance orchestration as likely referring here.
- `nstance-dev/terraform-aws-nstance` — AWS OpenTofu/Terraform module for Nstance.
- `nstance-dev/terraform-gcp-nstance` — Google Cloud OpenTofu/Terraform module for Nstance.
- `nstance-dev/website` — Nstance website.

### Easy OIDC

- `easy-oidc/easy-oidc` — minimal OIDC server used for authentication/OIDC scenarios.
- `easy-oidc/terraform-aws-easy-oidc` — AWS OpenTofu/Terraform module for Easy OIDC.
- `easy-oidc/website` — Easy OIDC website.

### puidv7

- `puidv7/puidv7-go` — Go implementation of prefixed UUIDv7 identifiers.
- `puidv7/puidv7-js` — JavaScript/TypeScript implementation of prefixed UUIDv7 identifiers.
- `puidv7/terraform-provider-puidv7` — OpenTofu/Terraform provider for generating/working with puidv7 identifiers.
- `puidv7/website` — puidv7 website.

## Example Cross-Repo Interpretation

If the user says something like:

> I'm working on the Podplane CLI and it needs to figure out how to orchestrate an Nstance server for the Netsy instance running in vmconfig.

Interpret it as:

- **Podplane CLI**: start in `~/Workspace/podplane/podplane` and that repo's `AGENTS.md`.
- **Nstance server orchestration**: inspect `~/Workspace/nstance-dev/nstance`, especially the server/orchestration/provider areas and that repo's `AGENTS.md`.
- **Netsy instance**: inspect `~/Workspace/netsy-dev/netsy`, especially server/runtime/configuration behavior and that repo's `AGENTS.md`.
- **running in vmconfig**: inspect `~/Workspace/podplane/vmconfig` and its `AGENTS.md` once present.

Do not assume all of the implementation belongs in this repository - it does not. This repo helps discover and coordinate the relevant repos; product code normally lives in the component repositories above.

## Working Across Repositories

- Before editing any component repository, read that repository's root `AGENTS.md` if present and follow its local build/test/style instructions.
- Also check that the git repository does not have any local staged/unstaged/untracked changes/files before creating/editing/deleting any files.
- Prefer the owning repository's Makefile commands over raw tool commands. For example, Netsy and Nstance both define project-specific `make build`, `make test`, and related commands.
- Keep changes scoped to the repository that owns the behavior. Only change multiple repos when the task truly crosses an API/contract boundary.
- When a task spans repos, identify the contract first: CLI command/API shape, config schema, protobuf/gRPC API, Terraform module input/output, object storage layout, or deployment convention.
- Verify in the smallest useful scope for each touched repo. If changing shared cross-repo behavior, run targeted checks in each affected repo.
- Do not vendor or copy code between repos unless explicitly asked; prefer stable interfaces and documented configuration/API contracts.
