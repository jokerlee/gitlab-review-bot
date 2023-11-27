package service

import (
	"github.com/jokerlee/gitlab-review-bot/internal/app/ds"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func (s *Service) commitsHandler(commit *ds.Commit) error {
	// fetch commit from repository
	old, err := s.r.CommitByID(commit.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch commit from repository, id:%s", commit.ID)
	}

	// if no changes, do nothing
	if old != nil && old.IsEqual(commit) {
		log.Debug().
			Int("project_id", commit.ProjectID).
			Str("id", commit.ID).
			Msg("commit skipped")
		return nil
	}

	// update (or create) it
	err = s.r.UpsertCommit(commit)
	if err != nil {
		return errors.Wrapf(err, "failed to update commit in repository, id:%s", commit.ID)
	}

	diffs, err := s.gitlab.GetCommitDiff(commit.ProjectID, commit.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to get diff of commit, project:%d, commit:%s", commit.ProjectID, commit.ID)
	}

	message := ComposeMessageForAI(commit.Title, commit.Message, diffs)
	reviewComment, err := s.openai.GenerateAICodeReviewComment(message)
	if err != nil {
		return errors.Wrap(err, "call openai failed")
	}
	log.Info().Msg(reviewComment)

	err = s.gitlab.AddCommentToCommit(commit.ProjectID, commit.ID, reviewComment)
	if err != nil {
		return errors.Wrap(err, "failed to add comment to merge request in repository")
	}

	log.Info().
		Int("project_id", commit.ProjectID).
		Str("iid", commit.ID).
		Str("url", commit.WebURL).
		Msg("AI review comment is added")

	return nil
}
