package app

import (
	"github.com/pkg/errors"

	"github.com/jokerlee/gitlab-review-bot/internal/app/ds"
	rd "github.com/jokerlee/gitlab-review-bot/internal/app/policy/reinventing-democracy"
	tlar "github.com/jokerlee/gitlab-review-bot/internal/app/policy/team-lead-always-right"
	"github.com/jokerlee/gitlab-review-bot/internal/app/service"
)

func (a *App) initPolicies() error {
	a.policies = make(map[ds.PolicyName]service.Policy)

	a.policies[rd.PolicyName] = rd.New(a.repository, a.gitlabClient)
	a.policies[tlar.PolicyName] = tlar.New(a.repository, a.gitlabClient)

	return nil
}

func (a *App) initService() error {
	var err error

	a.service, err = service.New(a.repository, a.gitlabClient, a.policies, a.slackClient, a.openaiClient)
	if err != nil {
		return errors.Wrap(err, "failed to init service")
	}

	return nil
}
