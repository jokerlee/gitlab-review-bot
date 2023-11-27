package gitlab

import (
	"github.com/jokerlee/gitlab-review-bot/internal/app/service"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"time"
)

func (c *Client) GetCommitDiff(projectID int, commitID string) (result []*service.Diff, err error) {
	c.rl.Take()
	diffs, _, err := c.gitlab.Commits.GetCommitDiff(
		projectID,
		commitID,
		&gitlab.GetCommitDiffOptions{
			Page:    1,
			PerPage: 20,
		})

	if err != nil {
		err = errors.Wrap(err, "error get diffs of the commitId")
		return
	}

	result = make([]*service.Diff, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, &service.Diff{
			Content:     diff.Diff,
			NewPath:     diff.NewPath,
			OldPath:     diff.OldPath,
			NewFile:     diff.NewFile,
			DeletedFile: diff.DeletedFile,
			RenamedFile: diff.RenamedFile,
		})
	}

	return
}

func (c *Client) AddCommentToCommit(projectID int, commitID string, comment string) error {
	c.rl.Take()
	var now = time.Now()
	_, _, err := c.gitlab.Discussions.CreateCommitDiscussion(
		projectID,
		commitID,
		&gitlab.CreateCommitDiscussionOptions{
			Body:      &comment,
			CreatedAt: &now,
		})

	if err != nil {
		return errors.Wrap(err, "error add comment to comment")
	}

	return nil
}
