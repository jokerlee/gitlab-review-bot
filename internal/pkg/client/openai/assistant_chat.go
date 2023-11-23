package openai

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"time"
)

const AssistantName = "Code Mentor"

func (c *Client) GenerateAICodeReviewComment(diff string) (string, error) {
	diff = truncate(diff, 16000)

	assistant, err := c.retrieveAssistant()
	if err != nil {
		return "", err
	}

	thread, err := c.openai.CreateThread(c.ctx, openai.ThreadRequest{
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
	run, err := c.openai.CreateRun(c.ctx, thread.ID, openai.RunRequest{
		AssistantID:  assistant.ID,
		Model:        &model,
		Instructions: &instruction,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to CreateRun from openai")
	}

	return c.waitRunToComplete(thread.ID, run.ID)
}

func (c *Client) retrieveAssistant() (assistant openai.Assistant, err error) {
	limit := 20
	order := "asc"
	resp, err := c.openai.ListAssistants(c.ctx, &limit, &order, nil, nil)
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
		assistant, err = c.createAssistant()
	}

	return assistant, nil
}

// createAssistant for first Run
func (c *Client) createAssistant() (assistant openai.Assistant, err error) {
	name := AssistantName
	description := "Code Review Master"
	instructions := "The GPT is designed to act as a code reviewer. " +
		"Its primary function is to assist users by identifying issues in their code. " +
		"It focuses on pinpointing naming inconsistencies, coding style breaches, concurrency pitfalls, " +
		"structural problems, duplicated code, cyclomatic complexity issues, logic errors, " +
		"and other code smells that could hinder maintainability and performance."

	return c.openai.CreateAssistant(c.ctx, openai.AssistantRequest{
		Model:        openai.GPT3Dot5Turbo16K,
		Name:         &name,
		Description:  &description,
		Instructions: &instructions,
	})
}

// wait until run completed or failed
func (c *Client) waitRunToComplete(threadId string, runId string) (string, error) {
	for {
		run, err := c.openai.RetrieveRun(c.ctx, threadId, runId)
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
			messages, err := c.openai.ListMessage(c.ctx, threadId, nil, nil, nil, nil)
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
