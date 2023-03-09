package main

import (
	"context"
	"github.com/google/go-github/v50/github"
)

func fetchPRDetails(prNumber int) (*github.PullRequest, error) {
	client := github.NewClient(nil)

	pr, _, err := client.PullRequests.Get(context.Background(), "linode", "linode-cli", prNumber)
	return pr, err
}
