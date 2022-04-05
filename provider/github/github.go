package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/aaomidi/virtual-branches-action/provider"
	"github.com/aaomidi/virtual-branches-action/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v42/github"
)

// Provider is the Provider specific to the GitHub provider
type Provider struct {
	GitHubClient *github.Client

	Owner      string
	Repository string
	Label      string
	Prefix     string

	RepositoryPath string
}

// GetConfigurations gets the virtual branch configurations from the GitHub Provider
func (g Provider) GetConfigurations(ctx context.Context) ([]provider.VirtualBranchConfig, error) {
	var allConfigs []provider.VirtualBranchConfig
	branches, err := g.getExistingBranches(ctx)
	if err != nil {
		return nil, err
	}

	issues, err := g.getAllIssues(ctx)
	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		config, err := processIssue(issue)
		if err != nil {
			// TODO what do we do with this error?
			fmt.Println(err)
			continue
		}

		err = validateConfiguration(config, branches)
		if err != nil {
			// TODO what about this one?
			fmt.Println(err)
			continue
		}

		allConfigs = append(allConfigs, config)
		spew.Dump(config)
	}
	return allConfigs, nil
}

func (g Provider) ApplyConfigurations(ctx context.Context, configs []provider.VirtualBranchConfig) ([]error, error) {
	for _, config := range configs {
		g.applyConfiguration(ctx, config)
	}
}

func (g Provider) applyConfiguration(ctx context.Context, vbConfig provider.VirtualBranchConfig) error {
	for _, track := range vbConfig.Track {
		ref := fmt.Sprintf("refs/remotes/orign/%s", track)
		opts := git.FetchOptions{
			RefSpecs: []config.RefSpec{config.RefSpec(ref)},
		}
		err := g.RepositoryClient.FetchContext(ctx, &opts)
		if err != nil {
			return err
		}

		g.RepositoryClient.CommitObjects()

	}
	return nil
}

func (g Provider) getAllIssues(ctx context.Context) ([]*github.Issue, error) {
	opt := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{g.Label},
	}

	var allIssues []*github.Issue
	for {
		issues, response, err := g.GitHubClient.Issues.ListByRepo(ctx, g.Owner, g.Repository, opt)
		if err != nil {
			return nil, fmt.Errorf("error retrieving issues from github: %w", err)
		}

		allIssues = append(allIssues, issues...)
		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return allIssues, nil
}

func (g Provider) getExistingBranches(ctx context.Context) (util.StringLookup, error) {
	branches, err := g.RepositoryClient.Branches()
	if err != nil {
		return nil, fmt.Errorf("error getting branches: %w", err)
	}
	allBranches := make(map[string]bool)
	err = branches.ForEach(func(reference *plumbing.Reference) error {
		branch := reference.Name().String()
		allBranches[branch] = true

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error iterating over branches branches: %w", err)
	}
	return allBranches, nil
}

func validateConfiguration(config provider.VirtualBranchConfig, branches util.StringLookup) (err error) {
	var searchBranches []string
	searchBranches = append(searchBranches, config.Base)
	searchBranches = append(searchBranches, config.Track...)
	err = validateBranchesExist(branches, searchBranches...)
	if err != nil {
		return
	}

	// We don't want to have branches that have `/` in them. That'd mess a lot of things up.
	if !util.ValidateTargetBranchName(config.Target) {
		//goland:noinspection GoErrorStringFormat
		err = fmt.Errorf("target branch name did not match the allowed characters of: a-z, A-Z, 0-9, _, -")
		return
	}

	return nil
}

func validateBranchesExist(branches util.StringLookup, searchBranches ...string) error {
	for _, branch := range searchBranches {
		if !branches[branch] {
			return fmt.Errorf("branch does not exist: %s", branch)
		}
	}

	return nil
}

type issueBody struct {
	Target string
	Base   string
	Track  []string
}

func processIssue(issue *github.Issue) (config provider.VirtualBranchConfig, err error) {
	issueBody := issueBody{}

	// Let's handle the use case where someone is using a codeblock.
	// The codeblock can look like '`', '```', or '```toml' followed by toml.
	// Let's trim all of that off.
	bodyStr := strings.Trim(issue.GetBody(), "`toml")
	meta, err := toml.Decode(bodyStr, &issueBody)
	if err != nil {
		return
	}

	err = isDefinedInMeta(meta, "Target", "Base", "Track")
	if err != nil {
		return
	}

	config = virtualBranchConfigFrom(issueBody)
	return
}

func isDefinedInMeta(metadata toml.MetaData, keys ...string) error {
	for _, key := range keys {
		if !metadata.IsDefined(key) {
			return fmt.Errorf("%s was not defined in configuration", key)
		}
	}
	return nil
}

func virtualBranchConfigFrom(issue issueBody) (config provider.VirtualBranchConfig) {
	config.Target = issue.Target
	config.Base = issue.Base
	config.Track = issue.Track

	return
}
