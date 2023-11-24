package service

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spatecon/gitlab-review-bot/internal/app/ds"
)

func (s *Service) mergeRequestsHandler(mr *ds.MergeRequest) error {
	// fetch MR from repository
	old, err := s.r.MergeRequestByID(mr.ID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch merge request from repository")
	}

	// if no changes, do nothing
	if old != nil && old.IsEqual(mr) {
		log.Debug().
			Int("project_id", mr.ProjectID).
			Int("iid", mr.IID).
			Msg("mr skipped")
		return nil
	}

	// enrich MR with approves
	approves, err := s.gitlab.MergeRequestApproves(mr.ProjectID, mr.IID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch merge request approves")
	}

	mr.Approves = approves

	// update (or create) it
	err = s.r.UpsertMergeRequest(mr)
	if err != nil {
		return errors.Wrap(err, "failed to update merge request in repository")
	}

	diff, err := s.gitlab.GetMergeRequestDiff(mr.ProjectID, mr.IID)
	if err != nil {
		return errors.Wrap(err, "failed to get diff of merge request")
	}
	log.Info().Msg(diff)

	reviewComment, err := s.openai.GenerateAICodeReviewComment(diff)
	if err != nil {
		return errors.Wrap(err, "call openai failed")
	}
	log.Info().Msg(reviewComment)

	err = s.gitlab.AddCommentToMergeRequests(mr.ProjectID, mr.IID, reviewComment)
	if err != nil {
		return errors.Wrap(err, "failed to add comment to merge request in repository")
	}

	log.Info().
		Int("project_id", mr.ProjectID).
		Int("iid", mr.IID).
		Str("url", mr.URL).
		Msg("mr updated or created")

	// process MR
	for _, team := range s.teams {
		if mr.CreatedAt != nil && mr.CreatedAt.Before(team.CreatedAt) {
			log.Info().Str("team_id", team.ID).Msg("skip team, mr created before team")
			continue
		}

		policy, ok := s.policies[team.Policy]
		if !ok {
			log.Error().
				Str("team", team.Name).
				Str("policy", string(team.Policy)).
				Msg("failed to process updates unknown policy")
			continue
		}

		err = policy.ProcessChanges(team, mr)
		if err != nil {
			log.Error().
				Err(err).
				Str("team", team.Name).
				Str("policy", string(team.Policy)).
				Msg("failed to process merge request")
		}

	}

	return nil
}

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
		return errors.Wrap(err, "failed to update merge request in repository")
	}

	diff, err := s.gitlab.GetCommitDiff(commit.ProjectID, commit.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get diff of merge request")
	}
	log.Info().Msg(diff)

	reviewComment, err := s.openai.GenerateAICodeReviewComment(diff)
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
		Msg("mr updated or created")

	return nil
}
