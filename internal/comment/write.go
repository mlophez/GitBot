package comment

import (
	comment "gitbot/internal/comment/types"
)

type CommentRepository interface {
	WriteComment(repo string, prId int, parentId int, msg string) error
}

type WriteCommentInPullRequestImpl func(prID int, cm comment.Comment, parent *comment.Comment) error

func WriteCommentInPullRequest(prID int, cm comment.Comment, parent *comment.Comment, commentRepo CommentRepository) error {
	if parent != nil {
		return commentRepo.WriteComment(cm.Repository(), prID, parent.ID(), cm.Message())
	}
	return commentRepo.WriteComment(cm.Repository(), prID, 0, cm.Message())
}

func WriteCommentInPullRequest2(commentRepo CommentRepository) WriteCommentInPullRequestImpl {
	return func(prID int, cm comment.Comment, parent *comment.Comment) error {
		if parent != nil {
			return commentRepo.WriteComment(cm.Repository(), prID, parent.ID(), cm.Message())
		}
		return commentRepo.WriteComment(cm.Repository(), prID, 0, cm.Message())
	}
}
