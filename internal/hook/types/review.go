package types

import (
	"errors"
)

type ReviewStatus struct {
	reviewersCount   int
	approvedCount    int
	changesRequested int
}

// NewReviewStatus crea un nuevo ReviewStatus con validación.
func NewReviewStatus(reviewersCount, approvedCount, changesRequested int) (*ReviewStatus, error) {
	if reviewersCount < 0 || approvedCount < 0 || changesRequested < 0 {
		return nil, errors.New("review counts cannot be negative")
	}
	if approvedCount > reviewersCount {
		return nil, errors.New("approved count cannot be greater than reviewers count")
	}
	return &ReviewStatus{
		reviewersCount:   reviewersCount,
		approvedCount:    approvedCount,
		changesRequested: changesRequested,
	}, nil
}

// Getters
//func (rs *ReviewStatus) ReviewersCount() int        { return rs.reviewersCount }
//func (rs *ReviewStatus) ApprovedCount() int         { return rs.approvedCount }
//func (rs *ReviewStatus) ChangesRequested() int      { return rs.changesRequested }
//func (rs *ReviewStatus) IsFullyApproved() bool      { return rs.approvedCount == rs.reviewersCount }
//func (rs *ReviewStatus) HasChangesRequested() bool  { return rs.changesRequested > 0 }


