package types

import (
	"errors"
)

//var (
//	ErrEmptyField     = errors.New("field must not be empty")
//	ErrFieldTooLong   = errors.New("field must not exceed 1024 characters")
//	ErrInvalidRepoURL = errors.New("link must be a valid URL")
//)

type PullRequest struct {
	id                int
	title             string
	sourceBranch      string
	targetBranch      string
	reviewStatus      *ReviewStatus
	commitsBehindBase int
	changedPaths      []string
}

// NewPullRequest crea una nueva PullRequest con validaciones.
func NewPullRequest(
	id int,
	title string,
	sourceBranch string,
	targetBranch string,
	reviewStatus *ReviewStatus,
	commitsBehindBase int,
	changedPaths []string,
) (*PullRequest, error) {
	if sourceBranch == targetBranch {
		return nil, errors.New("source and target branches cannot be the same")
	}

	return &PullRequest{
		id:                id,
		title:             title,
		sourceBranch:      sourceBranch,
		targetBranch:      targetBranch,
		reviewStatus:      reviewStatus,
		commitsBehindBase: commitsBehindBase,
		changedPaths:      changedPaths,
	}, nil
}

// Getters
// func (pr *PullRequest) ID() int                    { return pr.id }
// func (pr *PullRequest) Title() string              { return pr.title }
// func (pr *PullRequest) SourceBranch() string       { return pr.sourceBranch }
// func (pr *PullRequest) TargetBranch() string       { return pr.targetBranch }
// func (pr *PullRequest) ReviewStatus() *ReviewStatus { return pr.reviewStatus }
// func (pr *PullRequest) CommitsBehindBase() int     { return pr.commitsBehindBase }
// func (pr *PullRequest) ChangedPaths() []string     { return pr.changedPaths }
