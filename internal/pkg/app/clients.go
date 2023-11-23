package app

import (
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"net/http"
	"net/url"

	"github.com/spatecon/gitlab-review-bot/internal/pkg/client/gitlab"
	"github.com/spatecon/gitlab-review-bot/internal/pkg/client/slack"
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

	config := openai.DefaultConfig(a.cfg.OpenAIToken)
	if len(a.cfg.OpenAIProxyUrl) != 0 {
		proxyUrl, _ := url.Parse(a.cfg.OpenAIProxyUrl)
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}
	a.openaiClient = openai.NewClientWithConfig(config)

	return nil
}
