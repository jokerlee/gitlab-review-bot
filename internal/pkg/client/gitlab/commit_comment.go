package gitlab

import (
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"strings"
	"time"
)

func (c *Client) GetCommitDiff(projectID int, commitID string) (string, error) {
	c.rl.Take()
	diffs, _, err := c.gitlab.Commits.GetCommitDiff(
		projectID,
		commitID,
		&gitlab.GetCommitDiffOptions{
			Page:    1,
			PerPage: 20,
		})

	if err != nil {
		return "", errors.Wrap(err, "error get diffs of the commitId")
	}

	if len(diffs) == 0 {
		return "", nil
	}

	diffSlice := make([]string, 0, len(diffs))
	for _, diff := range diffs {
		diffSlice = append(diffSlice, diff.Diff)
	}

	return strings.Join(diffSlice, "\n"), nil
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
