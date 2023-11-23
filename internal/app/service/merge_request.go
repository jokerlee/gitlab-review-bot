package service

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/spatecon/gitlab-review-bot/internal/app/ds"
)

const AssistantName = "Code Mentor"

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
		//return nil
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

	reviewComment, err := chatWithAI(s.openai, diff)
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

func chatWithAI(aiClient *openai.Client, diff string) (string, error) {
	diff = truncate(diff, 16000)

	assistant, err := retrieveAssistant(aiClient)
	if err != nil {
		return "", err
	}

	thread, err := aiClient.CreateThread(context.Background(), openai.ThreadRequest{
		Messages: []openai.ThreadMessage{{
			Role:    openai.ChatMessageRoleUser,
			Content: diff,
		}},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to CreateThread from openai")
	}

	model := openai.GPT3Dot5Turbo16K
	instruction := "please review this code diff, give modification advice"
	run, err := aiClient.CreateRun(context.Background(), thread.ID, openai.RunRequest{
		AssistantID:  assistant.ID,
		Model:        &model,
		Instructions: &instruction,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to CreateRun from openai")
	}

	return waitRunToComplete(aiClient, thread.ID, run.ID)
}

func retrieveAssistant(aiClient *openai.Client) (assistant openai.Assistant, err error) {
	limit := 20
	order := "asc"
	resp, err := aiClient.ListAssistants(context.Background(), &limit, &order, nil, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to ListAssistants from openai")
		return
	}

	for _, item := range resp.Assistants {
		if *item.Name == AssistantName {
			assistant = item
		}
	}

	if assistant.ID == "" {
		assistant, err = createAssistant(aiClient)
	}

	return assistant, nil
}

// createAssistant for first Run
func createAssistant(aiClient *openai.Client) (assistant openai.Assistant, err error) {
	name := AssistantName
	description := "Code Review Master"
	instructions := "The GPT is designed to act as a code reviewer. " +
		"Its primary function is to assist users by identifying issues in their code. " +
		"It focuses on pinpointing naming inconsistencies, coding style breaches, concurrency pitfalls, " +
		"structural problems, duplicated code, cyclomatic complexity issues, logic errors, " +
		"and other code smells that could hinder maintainability and performance."

	return aiClient.CreateAssistant(context.Background(), openai.AssistantRequest{
		Model:        openai.GPT3Dot5Turbo16K,
		Name:         &name,
		Description:  &description,
		Instructions: &instructions,
	})
}

// wait until run completed or failed
func waitRunToComplete(aiClient *openai.Client, threadId string, runId string) (string, error) {
	for {
		run, err := aiClient.RetrieveRun(context.Background(), threadId, runId)
		if err != nil {
			return "", errors.Wrap(err, "failed to RetrieveRun from openai")
		}
		switch run.Status {
		case openai.RunStatusQueued:
		case openai.RunStatusInProgress:
		case openai.RunStatusCancelling:
			time.Sleep(1 * time.Second)
			// keep waiting
			continue
		case openai.RunStatusFailed:
		case openai.RunStatusExpired:
		case openai.RunStatusRequiresAction:
			return "", errors.Wrap(err, fmt.Sprintf("openai Run failed with status: %s", run.Status))
		case openai.RunStatusCompleted:
			messages, err := aiClient.ListMessage(context.Background(), threadId, nil, nil, nil, nil)
			if err != nil {
				return "", err
			} else {
				return messages.Messages[0].Content[0].Text.Value, nil
			}
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for i := range s {
		if n == 0 {
			return s[:i]
		}
		n--
	}
	return s
}
