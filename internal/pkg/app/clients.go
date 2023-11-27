package app

import (
	"github.com/jokerlee/gitlab-review-bot/internal/pkg/client/gitlab"
	"github.com/jokerlee/gitlab-review-bot/internal/pkg/client/openai"
	"github.com/jokerlee/gitlab-review-bot/internal/pkg/client/slack"
	"github.com/pkg/errors"
)

func (a *App) initClients() error {
	var err error

	a.gitlabClient, err = gitlab.New(a.ctx, a.cfg.GitlabServerUrl, a.cfg.GitlabToken)
	if err != nil {
		return errors.Wrap(err, "failed to init gitlab client")
	}

	a.slackClient, err = slack.New(a.ctx, a.cfg.SlackBotToken, a.cfg.SlackAppToken)
	if err != nil {
		return errors.Wrap(err, "failed to init slack client")
	}

	a.openaiClient, err = openai.New(a.ctx, a.cfg.OpenAIToken, a.cfg.OpenAIProxyUrl)
	return nil
}
