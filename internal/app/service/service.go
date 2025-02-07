//go:generate mockgen -source=service.go -destination=mocks/service.go -package=mocks -mock_names=Policy=Policy,SlackClient=SlackClient,Repository=Repository,GitlabClient=GitlabClient
package service

import (
	"context"
	"github.com/rs/zerolog/log"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"github.com/jokerlee/gitlab-review-bot/internal/app/ds"
	"github.com/jokerlee/gitlab-review-bot/internal/app/service/worker"
)

type Repository interface {
	Teams() ([]*ds.Team, error)
	Projects() ([]*ds.Project, error)
	MergeRequestByID(id int) (*ds.MergeRequest, error)
	MergeRequestsByProject(projectID int) ([]*ds.MergeRequest, error)
	MergeRequestsByAuthor(authorID []int) ([]*ds.MergeRequest, error)
	MergeRequestsByReviewer(reviewerID []int) ([]*ds.MergeRequest, error)
	UpsertMergeRequest(mr *ds.MergeRequest) error
	CommitByID(id string) (*ds.Commit, error)
	UpsertCommit(commit *ds.Commit) error
	UserBySlackID(slackID string) (*ds.User, *ds.Team, error)
}

type Diff struct {
	Content     string
	NewPath     string
	OldPath     string
	NewFile     bool
	RenamedFile bool
	DeletedFile bool
}

type GitlabClient interface {
	MergeRequestsByProject(projectID int, createdAfter time.Time) ([]*ds.MergeRequest, error)
	MergeRequestApproves(projectID int, iid int) ([]*ds.BasicUser, error)
	GetMergeRequestDiff(projectID int, iid int) ([]*Diff, error)
	AddCommentToMergeRequests(projectID int, iid int, comment string) error

	CommitsByProject(projectID int, createdAfter time.Time) ([]*ds.Commit, error)
	GetCommitDiff(projectID int, commitID string) ([]*Diff, error)
	AddCommentToCommit(projectID int, commitID string, comment string) error
}

type OpenAIClient interface {
	GenerateAICodeReviewComment(diff string) (string, error)
}

type SlackClient interface {
	worker.SlackClient
	Subscribe() (chan ds.UserEvent, error)
}

type Worker interface {
	Run()
	Close()
}

type Policy interface {
	// ProcessChanges may add new reviewers or do some actions
	ProcessChanges(team *ds.Team, mr *ds.MergeRequest) (err error)
	// ApprovedByUser checks if merge request is approved by passed users
	ApprovedByUser(team *ds.Team, mr *ds.MergeRequest, byAll ...*ds.BasicUser) bool
	// ApprovedByPolicy checks if merge request is approved by policy conditions
	ApprovedByPolicy(team *ds.Team, mr *ds.MergeRequest) bool
}

type Service struct {
	r        Repository
	gitlab   GitlabClient
	slack    SlackClient
	openai   OpenAIClient
	teams    []*ds.Team
	policies map[ds.PolicyName]Policy
	cron     *cron.Cron

	workers []Worker
}

func New(r Repository, g GitlabClient, p map[ds.PolicyName]Policy, slack SlackClient, openai OpenAIClient) (*Service, error) {
	svc := &Service{
		r:        r,
		gitlab:   g,
		slack:    slack,
		openai:   openai,
		teams:    nil,
		policies: p,
		cron:     nil,
		workers:  nil,
	}

	// TODO: team hot reload (just don't save it in service)
	err := svc.loadTeams()
	if err != nil {
		return nil, errors.Wrap(err, "failed to pre-cache teams")
	}

	err = svc.initNotifications()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init notifications")
	}

	return svc, nil
}

func (s *Service) Close() error {
	for _, wrk := range s.workers {
		wrk.Close()
	}

	cronCtx := s.cron.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-cronCtx.Done():
		return nil
	case <-ctx.Done():
		return errors.New("cron stopped dirty by timeout")
	}
}

func (s *Service) SubscribeOnSlack() error {
	events, err := s.slack.Subscribe()
	if err != nil {
		return errors.Wrap(err, "failed to subscribe on slack events")
	}

	wrk := worker.NewSlackWorker(s, s.r, s.slack, events)
	go wrk.Run()

	s.workers = append(s.workers, wrk)

	return nil
}

// SubscribeOnProjects Creates workers for each project and subscribe on merge requests changes
func (s *Service) SubscribeOnProjects(pullPeriod time.Duration) error {
	if pullPeriod < time.Second {
		return errors.Errorf("pull period is too small: %s", pullPeriod)
	}

	projects, err := s.r.Projects()
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		log.Warn().Msg("no project found")
	}

	for _, project := range projects {
		log.Info().Str("project_name", project.Name).Msg("init project watcher of")
		var wrk Worker

		wrk, err = worker.NewGitLabPuller(pullPeriod, project.CreatedAt, s.gitlab, s.mergeRequestsHandler, s.commitsHandler, project.ID)
		if err != nil {
			return errors.Wrap(err, "failed to create gitlab puller")
		}

		wrk.Run()

		s.workers = append(s.workers, wrk)
	}

	return nil
}
