.DEFAULT_GOAL := help

WORKSPACE ?= $(HOME)/Workspace
FETCH ?= 1

REPOS := \
	podplane/.github \
	podplane/workspace \
	podplane/podplane \
	podplane/vmconfig \
	podplane/components \
	podplane/kube2iam-binaries \
	podplane/templates \
	podplane/hello \
	podplane/workers \
	podplane/seedgen \
	podplane/seeds \
	podplane/website \
	podplane/terraform-provider-podplane \
	netsy-dev/.github \
	netsy-dev/conformance \
	netsy-dev/netsy \
	netsy-dev/website \
	nstance-dev/.github \
	nstance-dev/nstance \
	nstance-dev/terraform-aws-nstance \
	nstance-dev/terraform-gcp-nstance \
	nstance-dev/website \
	easy-oidc/.github \
	easy-oidc/easy-oidc \
	easy-oidc/terraform-aws-easy-oidc \
	easy-oidc/website \
	puidv7/.github \
	puidv7/puidv7-go \
	puidv7/puidv7-js \
	puidv7/terraform-provider-puidv7 \
	puidv7/website

.PHONY: help list status clone github-audit

help:
	@echo "Podplane workspace helper"
	@echo
	@echo "Usage:"
	@echo "  make list      Print the list of project-related Git repositories"
	@echo "  make status    Show Git status for every repository in your workspace directory"
	@echo "  make clone     Clone repositories not yet in your workspace directory"
	@echo "  make github-audit  Audit GitHub org and repository settings against github-policy.jsonc"
	@echo "  make help      Show this help screen"
	@echo
	@echo "Configuration:"
	@echo "  WORKSPACE=$(WORKSPACE)"
	@echo "  FETCH=$(FETCH)  # set FETCH=0 to skip git fetch during status"

list:
	@echo "Workspace: $(WORKSPACE)"
	@echo
	@echo "podplane"
	@echo "  Kubernetes distribution and PaaS."
	@for repo in $(REPOS); do \
		case "$$repo" in podplane/*) echo "  - git@github.com:$$repo.git" ;; esac; \
	done
	@echo
	@echo "netsy-dev"
	@echo "  Replicated key-value database backed by object storage."
	@for repo in $(REPOS); do \
		case "$$repo" in netsy-dev/*) echo "  - git@github.com:$$repo.git" ;; esac; \
	done
	@echo
	@echo "nstance-dev"
	@echo "  Next-generation VM autoscaling for AWS, Google Cloud, and Proxmox."
	@for repo in $(REPOS); do \
		case "$$repo" in nstance-dev/*) echo "  - git@github.com:$$repo.git" ;; esac; \
	done
	@echo
	@echo "easy-oidc"
	@echo "  Minimal OIDC server and deployment tooling."
	@for repo in $(REPOS); do \
		case "$$repo" in easy-oidc/*) echo "  - git@github.com:$$repo.git" ;; esac; \
	done
	@echo
	@echo "puidv7"
	@echo "  Prefixed UUIDv7 format and libraries."
	@for repo in $(REPOS); do \
		case "$$repo" in puidv7/*) echo "  - git@github.com:$$repo.git" ;; esac; \
	done

status:
	@if [ -t 1 ] && [ -z "$$NO_COLOR" ]; then \
		red=$$(printf '\033[31m'); \
		green=$$(printf '\033[32m'); \
		yellow=$$(printf '\033[33m'); \
		reset=$$(printf '\033[0m'); \
	else \
		red=""; green=""; yellow=""; reset=""; \
	fi; \
	echo "Workspace: $(WORKSPACE)"; \
	echo; \
	echo "Clone coverage"; \
	printf "%-42s %-10s %s\n" "REPOSITORY" "STATE" "PATH"; \
	missing=0; not_git=0; cloned=0; total=0; \
	for repo in $(REPOS); do \
		path="$(WORKSPACE)/$$repo"; \
		total=$$((total + 1)); \
		printf "%-42s " "$$repo"; \
		if [ ! -e "$$path" ]; then \
			missing=$$((missing + 1)); \
			printf "%s%-10s%s %s\n" "$$red" "missing" "$$reset" "$$path"; \
		elif [ ! -d "$$path/.git" ]; then \
			not_git=$$((not_git + 1)); \
			printf "%s%-10s%s %s\n" "$$red" "not-git" "$$reset" "$$path"; \
		else \
			cloned=$$((cloned + 1)); \
			printf "%s%-10s%s %s\n" "$$green" "cloned" "$$reset" "$$path"; \
		fi; \
	done; \
	printf "\n"; \
	if [ "$$missing" -eq 0 ] && [ "$$not_git" -eq 0 ]; then \
		printf "%sAll %s repositories are cloned.%s\n" "$$green" "$$total" "$$reset"; \
	else \
		printf "%s%s/%s cloned, %s missing, %s not Git repositories.%s\n" "$$red" "$$cloned" "$$total" "$$missing" "$$not_git" "$$reset"; \
	fi; \
	printf "\n"; \
	if [ "$(FETCH)" = "1" ]; then \
		echo "Fetching remotes"; \
		for repo in $(REPOS); do \
			path="$(WORKSPACE)/$$repo"; \
			if [ -d "$$path/.git" ]; then \
				printf "%-42s " "$$repo"; \
				if git -C "$$path" fetch --prune --quiet; then \
					printf "%s%s%s\n" "$$green" "ok" "$$reset"; \
				else \
					printf "%s%s%s\n" "$$red" "failed" "$$reset"; \
				fi; \
			fi; \
		done; \
	else \
		echo "Skipping fetch because FETCH=0"; \
	fi; \
	printf "\n"; \
	printf "%-64s%s\n" "Repository status" "(CHANGES: S=Staged U=Unstaged ?=Untracked)"; \
	printf "%-42s %-18s %-17s %-14s %s\n" "REPOSITORY" "BRANCH" "TRACKING" "SYNC" "CHANGES"; \
	for repo in $(REPOS); do \
		path="$(WORKSPACE)/$$repo"; \
		if [ ! -e "$$path" ]; then \
			printf "%-42s %-18s %-17s " "$$repo" "-" "-"; \
			printf "%s%-14s%s %s\n" "$$red" "missing" "$$reset" "-"; \
			continue; \
		fi; \
		if [ ! -d "$$path/.git" ]; then \
			printf "%-42s %-18s %-17s " "$$repo" "-" "-"; \
			printf "%s%-14s%s %s\n" "$$red" "not-git" "$$reset" "-"; \
			continue; \
		fi; \
		branch=$$(git -C "$$path" branch --show-current 2>/dev/null || true); \
		if [ -z "$$branch" ]; then \
			branch="detached:$$(git -C "$$path" rev-parse --short HEAD)"; \
		fi; \
		upstream=$$(git -C "$$path" rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null || true); \
		if [ -n "$$upstream" ]; then \
			track="$$upstream"; \
		elif git -C "$$path" rev-parse --verify --quiet "origin/$$branch" >/dev/null; then \
			track="origin/$$branch"; \
		else \
			track="-"; \
		fi; \
		if [ "$$track" != "-" ]; then \
			counts=$$(git -C "$$path" rev-list --left-right --count HEAD..."$$track" 2>/dev/null || echo "? ?"); \
			set -- $$counts; \
			ahead=$$1; behind=$$2; \
			if [ "$$behind" = "?" ] || [ "$$ahead" = "?" ]; then \
				sync="unknown"; sync_color="$$yellow"; \
			elif [ "$$behind" = "0" ] && [ "$$ahead" = "0" ]; then \
				sync="up-to-date"; sync_color="$$green"; \
			elif [ "$$behind" != "0" ]; then \
				sync="↓$$behind ↑$$ahead"; sync_color="$$red"; \
			else \
				sync="↑$$ahead"; sync_color="$$yellow"; \
			fi; \
		else \
			sync="no-track"; sync_color="$$yellow"; \
		fi; \
		staged=$$(git -C "$$path" diff --cached --name-only | wc -l | tr -d '[:space:]'); \
		unstaged=$$(git -C "$$path" diff --name-only | wc -l | tr -d '[:space:]'); \
		untracked=$$(git -C "$$path" ls-files --others --exclude-standard | wc -l | tr -d '[:space:]'); \
		if [ "$$staged" = "0" ] && [ "$$unstaged" = "0" ] && [ "$$untracked" = "0" ]; then \
			changes="clean"; \
		else \
			changes="dirty"; \
			if [ "$$staged" = "0" ]; then staged_color="$$green"; else staged_color="$$red"; fi; \
			if [ "$$unstaged" = "0" ]; then unstaged_color="$$green"; else unstaged_color="$$red"; fi; \
			if [ "$$untracked" = "0" ]; then untracked_color="$$green"; else untracked_color="$$red"; fi; \
		fi; \
		printf "%-42s %-18s %-17s " "$$repo" "$$branch" "$$track"; \
		printf "%s%-14s%s " "$$sync_color" "$$sync" "$$reset"; \
		if [ "$$changes" = "clean" ]; then \
			printf "%s%s%s\n" "$$green" "clean" "$$reset"; \
		else \
			printf "%sS:%s%s %sU:%s%s %s?:%s%s\n" "$$staged_color" "$$staged" "$$reset" "$$unstaged_color" "$$unstaged" "$$reset" "$$untracked_color" "$$untracked" "$$reset"; \
		fi; \
	done

clone:
	@for repo in $(REPOS); do \
		path="$(WORKSPACE)/$$repo"; \
		url="git@github.com:$$repo.git"; \
		if [ -e "$$path" ]; then \
			echo "skip: $$repo already exists at $$path"; \
			continue; \
		fi; \
		mkdir -p "$$(dirname "$$path")"; \
		echo "clone: $$url -> $$path"; \
		git clone "$$url" "$$path"; \
	done

github-audit:
	@command -v go >/dev/null || { echo "go is required" >&2; exit 2; }
	@command -v gh >/dev/null || { echo "gh is required; install GitHub CLI and run 'gh auth login'" >&2; exit 2; }
	@go run ./cmd/github-audit -policy github-policy.jsonc -makefile Makefile
