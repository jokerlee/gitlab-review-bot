package service

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spatecon/gitlab-review-bot/internal/app/ds"
)

func (s *Service) commitsHandler(commit *ds.Commit) error {
	// fetch MR from repository
	old, err := s.r.CommitByID(commit.ID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch commit from repository")
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
		return errors.Wrap(err, "failed to update commit in repository")
	}

	diff, err := s.gitlab.GetCommitDiff(commit.ProjectID, commit.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get diff of merge request")
	}
	fullDiff := commit.Title + "\n" + commit.Message + "\n" + diff
	log.Info().Msg(commit.Title + "\n" + commit.Message + "\n")

	reviewComment, err := s.openai.GenerateAICodeReviewComment(fullDiff)
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
