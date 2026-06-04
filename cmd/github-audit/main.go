package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type policy struct {
	Organizations organizationPolicy    `json:"organizations"`
	Repository    repositoryPolicy      `json:"repository"`
	Ruleset       rulesetPolicy         `json:"ruleset"`
	Repos         map[string]repoPolicy `json:"repos"`
}

type repoPolicy struct {
	Skip                 bool              `json:"skip"`
	DesiredDefaultBranch string            `json:"-"`
	Repository           *repositoryPolicy `json:"repository"`
	Ruleset              *rulesetPolicy    `json:"ruleset"`
}

type organizationPolicy struct {
	DCOApp           *dcoAppPolicy           `json:"dco_app"`
	RepositoryAccess *repositoryAccessPolicy `json:"repository_access"`
}

type dcoAppPolicy struct {
	Required            *bool  `json:"required"`
	Slug                string `json:"slug"`
	RepositorySelection string `json:"repository_selection"`
}

type repositoryAccessPolicy struct {
	Teams                    []teamAccessPolicy `json:"teams"`
	AllowOtherTeams          *bool              `json:"allow_other_teams"`
	AllowDirectCollaborators *bool              `json:"allow_direct_collaborators"`
}

type teamAccessPolicy struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Permission string `json:"permission"`
}

type repositoryPolicy struct {
	DefaultBranch string             `json:"default_branch"`
	Visibility    *string            `json:"visibility"`
	Features      featuresPolicy     `json:"features"`
	Releases      releasesPolicy     `json:"releases"`
	PullRequests  pullRequestsPolicy `json:"pull_requests"`
	Commits       commitsPolicy      `json:"commits"`
}

type featuresPolicy struct {
	HasWiki        *bool `json:"has_wiki"`
	HasIssues      *bool `json:"has_issues"`
	HasProjects    *bool `json:"has_projects"`
	HasDiscussions *bool `json:"has_discussions"`
}

type releasesPolicy struct {
	Immutable *bool `json:"immutable"`
}

type pullRequestsPolicy struct {
	CreationPolicy           *string `json:"creation_policy"`
	AllowMergeCommit         *bool   `json:"allow_merge_commit"`
	AllowSquashMerge         *bool   `json:"allow_squash_merge"`
	AllowRebaseMerge         *bool   `json:"allow_rebase_merge"`
	AllowAutoMerge           *bool   `json:"allow_auto_merge"`
	DeleteBranchOnMerge      *bool   `json:"delete_branch_on_merge"`
	AllowUpdateBranch        *bool   `json:"allow_update_branch"`
	MergeCommitTitle         *string `json:"merge_commit_title"`
	MergeCommitMessage       *string `json:"merge_commit_message"`
	SquashMergeCommitTitle   *string `json:"squash_merge_commit_title"`
	SquashMergeCommitMessage *string `json:"squash_merge_commit_message"`
}

type commitsPolicy struct {
	WebCommitSignoffRequired *bool `json:"web_commit_signoff_required"`
}

type rulesetPolicy struct {
	Required    *bool       `json:"required"`
	Name        string      `json:"name"`
	Target      string      `json:"target"`
	Enforcement string      `json:"enforcement"`
	IncludeRefs []string    `json:"include_refs"`
	Rules       rulesPolicy `json:"rules"`
}

type rulesPolicy struct {
	Deletion             bool                        `json:"deletion"`
	NonFastForward       bool                        `json:"non_fast_forward"`
	PullRequest          *pullRequestPolicy          `json:"pull_request"`
	RequiredStatusChecks *requiredStatusChecksPolicy `json:"required_status_checks"`
}

type pullRequestPolicy struct {
	RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
	DismissStaleReviewsOnPush    bool `json:"dismiss_stale_reviews_on_push"`
	RequireCodeOwnerReview       bool `json:"require_code_owner_review"`
	RequireLastPushApproval      bool `json:"require_last_push_approval"`
}

type requiredStatusChecksPolicy struct {
	Required bool     `json:"required"`
	Checks   []string `json:"checks"`
}

type repoInfo struct {
	FullName                  string `json:"full_name"`
	DefaultBranch             string `json:"default_branch"`
	Archived                  bool   `json:"archived"`
	Disabled                  bool   `json:"disabled"`
	Visibility                string `json:"visibility"`
	HasWiki                   bool   `json:"has_wiki"`
	HasIssues                 bool   `json:"has_issues"`
	HasProjects               bool   `json:"has_projects"`
	HasDiscussions            bool   `json:"has_discussions"`
	PullRequestCreationPolicy string `json:"pull_request_creation_policy"`
	AllowMergeCommit          bool   `json:"allow_merge_commit"`
	AllowSquashMerge          bool   `json:"allow_squash_merge"`
	AllowRebaseMerge          bool   `json:"allow_rebase_merge"`
	AllowAutoMerge            bool   `json:"allow_auto_merge"`
	DeleteBranchOnMerge       bool   `json:"delete_branch_on_merge"`
	AllowUpdateBranch         bool   `json:"allow_update_branch"`
	MergeCommitTitle          string `json:"merge_commit_title"`
	MergeCommitMessage        string `json:"merge_commit_message"`
	SquashMergeCommitTitle    string `json:"squash_merge_commit_title"`
	SquashMergeCommitMessage  string `json:"squash_merge_commit_message"`
	WebCommitSignoffRequired  bool   `json:"web_commit_signoff_required"`
}

type immutableReleases struct {
	Enabled bool `json:"enabled"`
}

type orgInstallations struct {
	Installations []orgInstallation `json:"installations"`
}

type orgInstallation struct {
	AppSlug             string `json:"app_slug"`
	RepositorySelection string `json:"repository_selection"`
	App                 struct {
		Slug string `json:"slug"`
	} `json:"app"`
}

type repoTeam struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Permission string `json:"permission"`
}

type repoCollaborator struct {
	Login string `json:"login"`
}

type ruleset struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Target      string     `json:"target"`
	Enforcement string     `json:"enforcement"`
	Conditions  conditions `json:"conditions"`
	Rules       []rule     `json:"rules"`
}

type conditions struct {
	RefName refNameCondition `json:"ref_name"`
}

type refNameCondition struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type rule struct {
	Type       string         `json:"type"`
	Parameters map[string]any `json:"parameters"`
}

// main parses command-line flags and runs the GitHub repository ruleset audit.
func main() {
	policyPath := flag.String("policy", "github-policy.jsonc", "path to GitHub audit policy")
	makefilePath := flag.String("makefile", "Makefile", "path to Makefile containing REPOS")
	flag.Parse()

	if err := run(*policyPath, *makefilePath); err != nil {
		fmt.Fprintf(os.Stderr, "github-audit: %v\n", err)
		os.Exit(2)
	}
}

// run loads policy and repository inventory, audits organizations and repositories, and prints the summary report.
func run(policyPath, makefilePath string) error {
	pol, err := loadPolicy(policyPath)
	if err != nil {
		return err
	}
	repos, err := loadRepos(makefilePath)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return fmt.Errorf("no repositories found in %s", makefilePath)
	}
	if err := checkGH(); err != nil {
		return err
	}

	orgs := orgsFromRepos(repos)
	repoGroups := reposByOrg(repos)
	orgFailures := 0
	orgErrors := 0
	for _, org := range orgs {
		issues, err := auditOrg(org, repoGroups[org], pol.Organizations)
		if err != nil {
			orgErrors++
			printResult("ERR", "org/"+org, []string{err.Error()})
			continue
		}
		if len(issues) == 0 {
			printResult("OK", "org/"+org, []string{"compliant"})
			continue
		}
		orgFailures++
		printResult("FAIL", "org/"+org, issues)
	}
	if len(orgs) > 0 {
		fmt.Println()
	}

	failures := 0
	errorsSeen := 0
	for _, repo := range repos {
		repoPol := mergedRepoPolicy(pol, repo)
		if repoPol.Skip {
			printResult("SKIP", repo, []string{"skipped by policy"})
			continue
		}

		issues, err := auditRepo(repo, repoPol)
		if err != nil {
			errorsSeen++
			printResult("ERR", repo, []string{err.Error()})
			continue
		}
		if len(issues) == 0 {
			printResult("OK", repo, []string{"compliant"})
			continue
		}
		failures++
		printResult("FAIL", repo, issues)
	}

	fmt.Println()
	fmt.Printf("Audited %d organizations: %d compliant, %d non-compliant, %d errors.\n", len(orgs), len(orgs)-orgFailures-orgErrors, orgFailures, orgErrors)
	fmt.Printf("Audited %d repositories: %d compliant, %d non-compliant, %d errors.\n", len(repos), len(repos)-failures-errorsSeen, failures, errorsSeen)
	if orgFailures > 0 || orgErrors > 0 || failures > 0 || errorsSeen > 0 {
		os.Exit(1)
	}
	return nil
}

// loadPolicy reads a JSONC policy file and fills default values for omitted fields.
func loadPolicy(path string) (policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return policy{}, fmt.Errorf("read policy: %w", err)
	}
	data, err = stripJSONComments(data)
	if err != nil {
		return policy{}, fmt.Errorf("parse policy comments: %w", err)
	}
	var pol policy
	if err := json.Unmarshal(data, &pol); err != nil {
		return policy{}, fmt.Errorf("parse policy: %w", err)
	}
	if pol.Repository.DefaultBranch == "" {
		pol.Repository.DefaultBranch = "main"
	}
	if pol.Organizations.DCOApp != nil {
		if pol.Organizations.DCOApp.Slug == "" {
			pol.Organizations.DCOApp.Slug = "dco"
		}
		if pol.Organizations.DCOApp.RepositorySelection == "" {
			pol.Organizations.DCOApp.RepositorySelection = "all"
		}
	}
	if pol.Organizations.RepositoryAccess != nil {
		for i := range pol.Organizations.RepositoryAccess.Teams {
			team := &pol.Organizations.RepositoryAccess.Teams[i]
			if team.Slug == "" {
				team.Slug = teamSlug(team.Name)
			}
			team.Permission = githubPermission(team.Permission)
		}
	}
	if pol.Ruleset.Target == "" {
		pol.Ruleset.Target = "branch"
	}
	if pol.Ruleset.Enforcement == "" {
		pol.Ruleset.Enforcement = "active"
	}
	if len(pol.Ruleset.IncludeRefs) == 0 {
		pol.Ruleset.IncludeRefs = []string{"~DEFAULT_BRANCH"}
	}
	if pol.Ruleset.Required == nil {
		pol.Ruleset.Required = boolPtr(true)
	}
	return pol, nil
}

// loadRepos extracts the managed GitHub repository list from the Makefile REPOS block.
func loadRepos(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read repos from %s: %w", path, err)
	}
	match := regexp.MustCompile(`(?ms)^REPOS\s*:=\s*\\\n(?P<body>.*?)(?:\n\S|\z)`).FindSubmatch(data)
	if match == nil {
		return nil, fmt.Errorf("REPOS block not found in %s", path)
	}
	body := string(match[1])
	var repos []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, "\\")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		repos = append(repos, line)
	}
	sort.Strings(repos)
	return repos, nil
}

// checkGH verifies that the GitHub CLI is installed and authenticated for github.com.
func checkGH() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("gh CLI is required; install it and run `gh auth login`")
	}
	cmd := exec.Command("gh", "auth", "status", "-h", "github.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh is not authenticated for github.com; run `gh auth login`: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// mergedRepoPolicy combines the global policy with any per-repository override.
func mergedRepoPolicy(pol policy, repo string) repoPolicy {
	ruleset := pol.Ruleset
	repository := pol.Repository
	merged := repoPolicy{DesiredDefaultBranch: repository.DefaultBranch, Repository: &repository, Ruleset: &ruleset}
	if override, ok := pol.Repos[repo]; ok {
		merged.Skip = override.Skip
		if override.Repository != nil {
			repository = mergeRepositoryPolicy(repository, *override.Repository)
			merged.Repository = &repository
			if repository.DefaultBranch != "" {
				merged.DesiredDefaultBranch = repository.DefaultBranch
			}
		}
		if override.Ruleset != nil {
			ruleset = mergeRulesetPolicy(ruleset, *override.Ruleset)
			merged.Ruleset = &ruleset
		}
	}
	return merged
}

// mergeRulesetPolicy overlays set ruleset settings from override onto base.
func mergeRulesetPolicy(base, override rulesetPolicy) rulesetPolicy {
	if override.Required != nil {
		base.Required = override.Required
	}
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.Target != "" {
		base.Target = override.Target
	}
	if override.Enforcement != "" {
		base.Enforcement = override.Enforcement
	}
	if len(override.IncludeRefs) > 0 {
		base.IncludeRefs = override.IncludeRefs
	}
	if override.Rules.PullRequest != nil {
		base.Rules.PullRequest = override.Rules.PullRequest
	}
	if override.Rules.RequiredStatusChecks != nil {
		base.Rules.RequiredStatusChecks = override.Rules.RequiredStatusChecks
	}
	return base
}

// mergeRepositoryPolicy overlays non-nil repository settings from override onto base.
func mergeRepositoryPolicy(base, override repositoryPolicy) repositoryPolicy {
	if override.DefaultBranch != "" {
		base.DefaultBranch = override.DefaultBranch
	}
	if override.Visibility != nil {
		base.Visibility = override.Visibility
	}
	if override.Features.HasWiki != nil {
		base.Features.HasWiki = override.Features.HasWiki
	}
	if override.Features.HasIssues != nil {
		base.Features.HasIssues = override.Features.HasIssues
	}
	if override.Features.HasProjects != nil {
		base.Features.HasProjects = override.Features.HasProjects
	}
	if override.Features.HasDiscussions != nil {
		base.Features.HasDiscussions = override.Features.HasDiscussions
	}
	if override.Releases.Immutable != nil {
		base.Releases.Immutable = override.Releases.Immutable
	}
	if override.PullRequests.CreationPolicy != nil {
		base.PullRequests.CreationPolicy = override.PullRequests.CreationPolicy
	}
	if override.PullRequests.AllowMergeCommit != nil {
		base.PullRequests.AllowMergeCommit = override.PullRequests.AllowMergeCommit
	}
	if override.PullRequests.AllowSquashMerge != nil {
		base.PullRequests.AllowSquashMerge = override.PullRequests.AllowSquashMerge
	}
	if override.PullRequests.AllowRebaseMerge != nil {
		base.PullRequests.AllowRebaseMerge = override.PullRequests.AllowRebaseMerge
	}
	if override.PullRequests.AllowAutoMerge != nil {
		base.PullRequests.AllowAutoMerge = override.PullRequests.AllowAutoMerge
	}
	if override.PullRequests.DeleteBranchOnMerge != nil {
		base.PullRequests.DeleteBranchOnMerge = override.PullRequests.DeleteBranchOnMerge
	}
	if override.PullRequests.AllowUpdateBranch != nil {
		base.PullRequests.AllowUpdateBranch = override.PullRequests.AllowUpdateBranch
	}
	if override.PullRequests.MergeCommitTitle != nil {
		base.PullRequests.MergeCommitTitle = override.PullRequests.MergeCommitTitle
	}
	if override.PullRequests.MergeCommitMessage != nil {
		base.PullRequests.MergeCommitMessage = override.PullRequests.MergeCommitMessage
	}
	if override.PullRequests.SquashMergeCommitTitle != nil {
		base.PullRequests.SquashMergeCommitTitle = override.PullRequests.SquashMergeCommitTitle
	}
	if override.PullRequests.SquashMergeCommitMessage != nil {
		base.PullRequests.SquashMergeCommitMessage = override.PullRequests.SquashMergeCommitMessage
	}
	if override.Commits.WebCommitSignoffRequired != nil {
		base.Commits.WebCommitSignoffRequired = override.Commits.WebCommitSignoffRequired
	}
	return base
}

// auditRepo fetches repository metadata and rulesets, then returns policy violations for one repository.
func auditRepo(repo string, pol repoPolicy) ([]string, error) {
	info, err := ghJSON[repoInfo]("/repos/" + repo)
	if err != nil {
		return nil, err
	}
	var issues []string
	if info.Archived {
		issues = append(issues, "repository is archived")
	}
	if info.Disabled {
		issues = append(issues, "repository is disabled")
	}
	if info.DefaultBranch != pol.DesiredDefaultBranch {
		issues = append(issues, fmt.Sprintf("default branch is %q, want %q", info.DefaultBranch, pol.DesiredDefaultBranch))
	}
	if pol.Repository != nil {
		issues = append(issues, auditRepositorySettings(repo, info, *pol.Repository)...)
	}
	if pol.Ruleset != nil && pol.Ruleset.Required != nil && !*pol.Ruleset.Required {
		return issues, nil
	}

	rulesets, err := ghJSON[[]ruleset]("/repos/" + repo + "/rulesets?includes_parents=true")
	if err != nil {
		if isRulesetsUnavailable(err) {
			issues = append(issues, "repository rulesets are unavailable for this repository or plan")
			return issues, nil
		}
		return nil, err
	}
	for i := range rulesets {
		if rulesets[i].ID != 0 && len(rulesets[i].Rules) == 0 {
			detail, err := ghJSON[ruleset](fmt.Sprintf("/repos/%s/rulesets/%d", repo, rulesets[i].ID))
			if err == nil {
				rulesets[i] = detail
			}
		}
	}

	bestIssues := []string{"no matching active branch ruleset protects the default branch"}
	for _, rs := range rulesets {
		rsIssues := auditRuleset(rs, info.DefaultBranch, *pol.Ruleset)
		if len(rsIssues) == 0 {
			return issues, nil
		}
		if len(bestIssues) == 1 || len(rsIssues) < len(bestIssues) {
			bestIssues = rsIssues
		}
	}
	issues = append(issues, bestIssues...)
	return issues, nil
}

// auditOrg checks organization-level GitHub App installation and repository access requirements.
func auditOrg(org string, repos []string, pol organizationPolicy) ([]string, error) {
	var issues []string
	appIssues, err := auditOrgApp(org, pol)
	if err != nil {
		return nil, err
	}
	issues = append(issues, appIssues...)
	accessIssues, err := auditOrgRepositoryAccess(repos, pol.RepositoryAccess)
	if err != nil {
		return nil, err
	}
	issues = append(issues, accessIssues...)
	return issues, nil
}

// auditOrgApp checks organization-level GitHub App installation requirements.
func auditOrgApp(org string, pol organizationPolicy) ([]string, error) {
	if pol.DCOApp == nil || pol.DCOApp.Required == nil || !*pol.DCOApp.Required {
		return nil, nil
	}
	installations, err := ghJSON[orgInstallations]("/orgs/" + org + "/installations")
	if err != nil {
		return nil, err
	}
	for _, installation := range installations.Installations {
		slug := installation.AppSlug
		if slug == "" {
			slug = installation.App.Slug
		}
		if slug != pol.DCOApp.Slug {
			continue
		}
		if installation.RepositorySelection != pol.DCOApp.RepositorySelection {
			return []string{fmt.Sprintf("GitHub App %q repository_selection is %q, want %q", pol.DCOApp.Slug, installation.RepositorySelection, pol.DCOApp.RepositorySelection)}, nil
		}
		return nil, nil
	}
	return []string{fmt.Sprintf("GitHub App %q is not installed", pol.DCOApp.Slug)}, nil
}

// auditOrgRepositoryAccess checks each managed repository has only the policy-allowed direct access grants.
func auditOrgRepositoryAccess(repos []string, pol *repositoryAccessPolicy) ([]string, error) {
	if pol == nil {
		return nil, nil
	}
	wantTeams := map[string]teamAccessPolicy{}
	for _, team := range pol.Teams {
		wantTeams[team.Slug] = team
	}
	allowOtherTeams := pol.AllowOtherTeams != nil && *pol.AllowOtherTeams
	allowDirectCollaborators := pol.AllowDirectCollaborators != nil && *pol.AllowDirectCollaborators

	var issues []string
	for _, repo := range repos {
		teams, err := ghJSON[[]repoTeam]("/repos/" + repo + "/teams")
		if err != nil {
			return nil, err
		}
		gotTeams := map[string]repoTeam{}
		for _, team := range teams {
			gotTeams[team.Slug] = team
		}
		for slug, want := range wantTeams {
			got, ok := gotTeams[slug]
			if !ok {
				issues = append(issues, fmt.Sprintf("%s: missing team %q with %s access", repo, teamLabel(want), policyPermission(want.Permission)))
				continue
			}
			if got.Permission != want.Permission {
				issues = append(issues, fmt.Sprintf("%s: team %q has %s access, want %s", repo, teamLabel(want), policyPermission(got.Permission), policyPermission(want.Permission)))
			}
		}
		if !allowOtherTeams {
			for _, got := range teams {
				if _, ok := wantTeams[got.Slug]; ok {
					continue
				}
				issues = append(issues, fmt.Sprintf("%s: unexpected team %q has %s access", repo, got.Name, policyPermission(got.Permission)))
			}
		}
		if !allowDirectCollaborators {
			collaborators, err := ghJSON[[]repoCollaborator]("/repos/" + repo + "/collaborators?affiliation=direct")
			if err != nil {
				return nil, err
			}
			for _, collaborator := range collaborators {
				issues = append(issues, fmt.Sprintf("%s: unexpected direct collaborator %q", repo, collaborator.Login))
			}
		}
	}
	return issues, nil
}

// auditRepositorySettings compares repository metadata and ancillary endpoints against policy.
func auditRepositorySettings(repo string, info repoInfo, pol repositoryPolicy) []string {
	var issues []string
	issues = appendStringIssue(issues, "repository.visibility", info.Visibility, pol.Visibility)
	issues = appendBoolIssue(issues, "repository.features.has_wiki", info.HasWiki, pol.Features.HasWiki)
	issues = appendBoolIssue(issues, "repository.features.has_issues", info.HasIssues, pol.Features.HasIssues)
	issues = appendBoolIssue(issues, "repository.features.has_projects", info.HasProjects, pol.Features.HasProjects)
	issues = appendBoolIssue(issues, "repository.features.has_discussions", info.HasDiscussions, pol.Features.HasDiscussions)
	issues = appendStringIssue(issues, "repository.pull_requests.creation_policy", info.PullRequestCreationPolicy, pol.PullRequests.CreationPolicy)
	issues = appendBoolIssue(issues, "repository.pull_requests.allow_merge_commit", info.AllowMergeCommit, pol.PullRequests.AllowMergeCommit)
	issues = appendBoolIssue(issues, "repository.pull_requests.allow_squash_merge", info.AllowSquashMerge, pol.PullRequests.AllowSquashMerge)
	issues = appendBoolIssue(issues, "repository.pull_requests.allow_rebase_merge", info.AllowRebaseMerge, pol.PullRequests.AllowRebaseMerge)
	issues = appendBoolIssue(issues, "repository.pull_requests.allow_auto_merge", info.AllowAutoMerge, pol.PullRequests.AllowAutoMerge)
	issues = appendBoolIssue(issues, "repository.pull_requests.delete_branch_on_merge", info.DeleteBranchOnMerge, pol.PullRequests.DeleteBranchOnMerge)
	issues = appendBoolIssue(issues, "repository.pull_requests.allow_update_branch", info.AllowUpdateBranch, pol.PullRequests.AllowUpdateBranch)
	issues = appendStringIssue(issues, "repository.pull_requests.merge_commit_title", info.MergeCommitTitle, pol.PullRequests.MergeCommitTitle)
	issues = appendStringIssue(issues, "repository.pull_requests.merge_commit_message", info.MergeCommitMessage, pol.PullRequests.MergeCommitMessage)
	issues = appendStringIssue(issues, "repository.pull_requests.squash_merge_commit_title", info.SquashMergeCommitTitle, pol.PullRequests.SquashMergeCommitTitle)
	issues = appendStringIssue(issues, "repository.pull_requests.squash_merge_commit_message", info.SquashMergeCommitMessage, pol.PullRequests.SquashMergeCommitMessage)
	issues = appendBoolIssue(issues, "repository.commits.web_commit_signoff_required", info.WebCommitSignoffRequired, pol.Commits.WebCommitSignoffRequired)
	if pol.Releases.Immutable != nil {
		immutable, err := immutableReleasesEnabled(repo)
		if err != nil {
			issues = append(issues, "repository.releases.immutable: "+err.Error())
		} else {
			issues = appendBoolIssue(issues, "repository.releases.immutable", immutable, pol.Releases.Immutable)
		}
	}
	return issues
}

// auditRuleset compares a GitHub ruleset against the desired branch ruleset policy.
func auditRuleset(rs ruleset, defaultBranch string, pol rulesetPolicy) []string {
	var issues []string
	label := rs.Name
	if label == "" {
		label = fmt.Sprintf("ruleset %d", rs.ID)
	}
	if pol.Name != "" && rs.Name != pol.Name {
		issues = append(issues, fmt.Sprintf("%s: name is %q, want %q", label, rs.Name, pol.Name))
	}
	if rs.Target != pol.Target {
		issues = append(issues, fmt.Sprintf("%s: target is %q, want %q", label, rs.Target, pol.Target))
	}
	if rs.Enforcement != pol.Enforcement {
		issues = append(issues, fmt.Sprintf("%s: enforcement is %q, want %q", label, rs.Enforcement, pol.Enforcement))
	}
	if !includesDefaultBranch(rs.Conditions.RefName.Include, defaultBranch, pol.IncludeRefs) {
		issues = append(issues, fmt.Sprintf("%s: does not include the default branch", label))
	}
	if excludesDefaultBranch(rs.Conditions.RefName.Exclude, defaultBranch) {
		issues = append(issues, fmt.Sprintf("%s: excludes the default branch", label))
	}

	rules := map[string]rule{}
	for _, r := range rs.Rules {
		rules[r.Type] = r
	}
	if pol.Rules.Deletion {
		if _, ok := rules["deletion"]; !ok {
			issues = append(issues, fmt.Sprintf("%s: missing deletion rule", label))
		}
	}
	if pol.Rules.NonFastForward {
		if _, ok := rules["non_fast_forward"]; !ok {
			issues = append(issues, fmt.Sprintf("%s: missing non_fast_forward rule", label))
		}
	}
	if pol.Rules.PullRequest != nil {
		pr, ok := rules["pull_request"]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: missing pull_request rule", label))
		} else {
			issues = append(issues, auditPullRequestRule(label, pr.Parameters, *pol.Rules.PullRequest)...)
		}
	}
	if pol.Rules.RequiredStatusChecks != nil {
		rsc, ok := rules["required_status_checks"]
		if pol.Rules.RequiredStatusChecks.Required && !ok {
			issues = append(issues, fmt.Sprintf("%s: missing required_status_checks rule", label))
		} else if ok {
			issues = append(issues, auditRequiredStatusChecksRule(label, rsc.Parameters, *pol.Rules.RequiredStatusChecks)...)
		}
	}
	return issues
}

// auditPullRequestRule compares pull request rule parameters against policy expectations.
func auditPullRequestRule(label string, params map[string]any, pol pullRequestPolicy) []string {
	checks := []struct {
		name string
		want any
	}{
		{"required_approving_review_count", float64(pol.RequiredApprovingReviewCount)},
		{"dismiss_stale_reviews_on_push", pol.DismissStaleReviewsOnPush},
		{"require_code_owner_review", pol.RequireCodeOwnerReview},
		{"require_last_push_approval", pol.RequireLastPushApproval},
	}
	var issues []string
	for _, check := range checks {
		got, ok := params[check.name]
		if !ok || got != check.want {
			issues = append(issues, fmt.Sprintf("%s: pull_request.%s is %v, want %v", label, check.name, got, check.want))
		}
	}
	return issues
}

// auditRequiredStatusChecksRule verifies that all policy-required status checks are present.
func auditRequiredStatusChecksRule(label string, params map[string]any, pol requiredStatusChecksPolicy) []string {
	if len(pol.Checks) == 0 {
		return nil
	}
	actual := map[string]bool{}
	if raw, ok := params["required_status_checks"].([]any); ok {
		for _, item := range raw {
			switch v := item.(type) {
			case string:
				actual[v] = true
			case map[string]any:
				if context, ok := v["context"].(string); ok {
					actual[context] = true
				}
			}
		}
	}
	var issues []string
	for _, want := range pol.Checks {
		if !actual[want] {
			issues = append(issues, fmt.Sprintf("%s: missing required status check %q", label, want))
		}
	}
	return issues
}

// appendBoolIssue appends a standard mismatch message when a boolean policy is set and not met.
func appendBoolIssue(issues []string, name string, got bool, want *bool) []string {
	if want == nil || got == *want {
		return issues
	}
	return append(issues, fmt.Sprintf("%s is %t, want %t", name, got, *want))
}

// appendStringIssue appends a standard mismatch message when a string policy is set and not met.
func appendStringIssue(issues []string, name string, got string, want *string) []string {
	if want == nil || got == *want {
		return issues
	}
	return append(issues, fmt.Sprintf("%s is %q, want %q", name, got, *want))
}

// boolPtr returns a pointer to value for defaulted optional policy booleans.
func boolPtr(value bool) *bool {
	return &value
}

// teamSlug derives GitHub's default team slug from a display name.
func teamSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, "-")
}

// githubPermission normalizes human policy names to GitHub API permission names.
func githubPermission(permission string) string {
	switch permission {
	case "read":
		return "pull"
	case "write":
		return "push"
	default:
		return permission
	}
}

// policyPermission renders GitHub API permission names as human policy names.
func policyPermission(permission string) string {
	switch permission {
	case "pull":
		return "read"
	case "push":
		return "write"
	default:
		return permission
	}
}

// teamLabel returns a stable human-readable team label for audit messages.
func teamLabel(team teamAccessPolicy) string {
	if team.Name != "" {
		return team.Name
	}
	return team.Slug
}

// orgsFromRepos returns sorted unique GitHub organization names from owner/repo entries.
func orgsFromRepos(repos []string) []string {
	seen := map[string]bool{}
	for _, repo := range repos {
		owner, _, ok := strings.Cut(repo, "/")
		if ok && owner != "" {
			seen[owner] = true
		}
	}
	orgs := make([]string, 0, len(seen))
	for org := range seen {
		orgs = append(orgs, org)
	}
	sort.Strings(orgs)
	return orgs
}

// reposByOrg groups sorted owner/repo entries by GitHub organization.
func reposByOrg(repos []string) map[string][]string {
	groups := map[string][]string{}
	for _, repo := range repos {
		owner, _, ok := strings.Cut(repo, "/")
		if ok && owner != "" {
			groups[owner] = append(groups[owner], repo)
		}
	}
	return groups
}

// printResult prints one grouped audit result immediately after that subject is checked.
func printResult(status, subject string, messages []string) {
	fmt.Printf("%s %s %s\n", statusIcon(status), status, subject)
	if status == "OK" && len(messages) == 1 && messages[0] == "compliant" {
		return
	}
	for _, message := range messages {
		fmt.Printf("  - %s\n", message)
	}
}

// statusIcon returns the display icon for a grouped audit status.
func statusIcon(status string) string {
	switch status {
	case "OK":
		return "✅"
	case "FAIL", "ERR":
		return "❌"
	case "SKIP":
		return "⏭️"
	default:
		return "•"
	}
}

// immutableReleasesEnabled reports whether release immutability is enabled, treating GitHub's 404 as disabled.
func immutableReleasesEnabled(repo string) (bool, error) {
	value, found, err := ghJSONOptional[immutableReleases]("/repos/" + repo + "/immutable-releases")
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	return value.Enabled, nil
}

// isRulesetsUnavailable reports whether GitHub says repository rulesets are not available for this repository.
func isRulesetsUnavailable(err error) bool {
	text := err.Error()
	return strings.Contains(text, "HTTP 403") && strings.Contains(text, "Upgrade to GitHub Pro")
}

// includesDefaultBranch reports whether ruleset include patterns cover the repository default branch.
func includesDefaultBranch(includes []string, defaultBranch string, policyIncludes []string) bool {
	allowed := map[string]bool{}
	for _, ref := range policyIncludes {
		allowed[ref] = true
		if ref == "~DEFAULT_BRANCH" {
			allowed["refs/heads/"+defaultBranch] = true
			allowed[defaultBranch] = true
		}
	}
	for _, ref := range includes {
		if allowed[ref] || ref == "~ALL" || ref == "refs/heads/"+defaultBranch || ref == defaultBranch {
			return true
		}
	}
	return false
}

// excludesDefaultBranch reports whether ruleset exclude patterns remove the repository default branch.
func excludesDefaultBranch(excludes []string, defaultBranch string) bool {
	for _, ref := range excludes {
		if ref == "~DEFAULT_BRANCH" || ref == "refs/heads/"+defaultBranch || ref == defaultBranch || ref == "~ALL" {
			return true
		}
	}
	return false
}

// ghJSON calls a read-only GitHub API endpoint with gh and decodes its JSON response.
func ghJSON[T any](endpoint string) (T, error) {
	var zero T
	cmd := exec.Command("gh", "api", "--paginate", endpoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return zero, fmt.Errorf("gh api %s failed: %s", endpoint, strings.TrimSpace(string(out)))
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var value T
	if err := dec.Decode(&value); err != nil {
		return zero, fmt.Errorf("decode gh api %s: %w", endpoint, err)
	}
	return value, nil
}

// ghJSONOptional calls a GitHub API endpoint where a 404 means the feature is disabled or absent.
func ghJSONOptional[T any](endpoint string) (T, bool, error) {
	var zero T
	cmd := exec.Command("gh", "api", endpoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if strings.Contains(text, "HTTP 404") || strings.Contains(strings.ToLower(text), "not found") {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("gh api %s failed: %s", endpoint, text)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var value T
	if err := dec.Decode(&value); err != nil {
		return zero, false, fmt.Errorf("decode gh api %s: %w", endpoint, err)
	}
	return value, true, nil
}

// stripJSONComments removes line and block comments from JSONC while preserving string contents.
func stripJSONComments(data []byte) ([]byte, error) {
	var out bytes.Buffer
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		ch := data[i]
		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && i+1 < len(data) {
			next := data[i+1]
			if next == '/' {
				i += 2
				for i < len(data) && data[i] != '\n' {
					i++
				}
				if i < len(data) {
					out.WriteByte('\n')
				}
				continue
			}
			if next == '*' {
				i += 2
				closed := false
				for i+1 < len(data) {
					if data[i] == '*' && data[i+1] == '/' {
						closed = true
						i++
						break
					}
					if data[i] == '\n' {
						out.WriteByte('\n')
					}
					i++
				}
				if !closed {
					return nil, errors.New("unterminated block comment")
				}
				continue
			}
		}
		out.WriteByte(ch)
	}
	if inString {
		return nil, errors.New("unterminated string")
	}
	return out.Bytes(), nil
}
