package app

import (
	"context"
	"github.com/jokerlee/gitlab-review-bot/internal/pkg/client/openai"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/jokerlee/gitlab-review-bot/internal/app/ds"
	"github.com/jokerlee/gitlab-review-bot/internal/app/repository"
	"github.com/jokerlee/gitlab-review-bot/internal/app/service"
	"github.com/jokerlee/gitlab-review-bot/internal/pkg/client/gitlab"
	"github.com/jokerlee/gitlab-review-bot/internal/pkg/client/slack"
)

type App struct {
	logger zerolog.Logger
	cfg    Config

	mongoClient *mongo.Client
	repository  *repository.Repository

	gitlabClient *gitlab.Client
	slackClient  *slack.Client
	openaiClient *openai.Client

	policies map[ds.PolicyName]service.Policy
	service  *service.Service

	// graceful shutdown
	ctx      context.Context
	closeCtx func()
}

func New(configPath string) (*App, error) {
	app := &App{}

	app.ctx, app.closeCtx = context.WithCancel(context.Background())

	err := app.initConfig(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}

	app.initLogger()

	err = app.initRepository()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init repository")
	}

	err = app.initClients()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init clients")
	}

	err = app.initPolicies()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init policies")
	}

	err = app.initService()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init service")
	}

	return app, nil
}

func (a *App) Run() error {
	var err error

	a.logger.Info().Msg("app started")

	//err = a.service.SubscribeOnSlack()
	//if err != nil {
	//	return errors.Wrap(err, "failed to subscribe on slack events")
	//}

	err = a.service.SubscribeOnProjects(a.cfg.PullPeriod)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe on projects")
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	<-ch
	a.closer()

	return nil
}
