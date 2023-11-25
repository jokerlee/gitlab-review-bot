package gitlab

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spatecon/gitlab-review-bot/internal/app/ds"
	"github.com/xanzy/go-gitlab"
	"time"
)

func (c *Client) CommitsByProject(projectID int, createdAfter time.Time) ([]*ds.Commit, error) {
	allCommits := make([]*ds.Commit, 0, perPage)
	withStats := true
	for i := 1; i <= maxPages; i++ {
		log.Trace().Msg("fetching commits")
		c.rl.Take()
		// docs: https://docs.gitlab.com/ee/api/commits.html
		commits, resp, err := c.gitlab.Commits.ListCommits(
			projectID,
			&gitlab.ListCommitsOptions{
				WithStats: &withStats,
				ListOptions: gitlab.ListOptions{
					Page:    i,
					PerPage: perPage,
				},
			},
			gitlab.WithContext(c.ctx))
		if err != nil {
			return nil, errors.Wrap(err, "error getting commits")
		}

		for _, commit := range commits {
			allCommits = append(allCommits, commitConvert(commit, projectID))
		}

		if resp.NextPage == 0 {
			break
		}
	}

	return allCommits, nil
}

func commitConvert(req *gitlab.Commit, projectID int) *ds.Commit {
	var stats *ds.CommitStats
	if req.Stats != nil {
		stats = &ds.CommitStats{
			Additions: req.Stats.Additions,
			Deletions: req.Stats.Deletions,
			Total:     req.Stats.Total,
		}
	}

	return &ds.Commit{
		ID:             req.ID,
		ShortID:        req.ShortID,
		Title:          req.Title,
		AuthorName:     req.AuthorName,
		AuthorEmail:    req.AuthorEmail,
		AuthoredDate:   req.AuthoredDate,
		CommitterName:  req.CommitterName,
		CommitterEmail: req.CommitterEmail,
		CommittedDate:  req.CommittedDate,
		CreatedAt:      req.CreatedAt,
		Message:        req.Message,
		ParentIDs:      req.ParentIDs,
		ProjectID:      projectID,
		Trailers:       req.Trailers,
		WebURL:         req.WebURL,
		Stats:          stats,
	}
}
