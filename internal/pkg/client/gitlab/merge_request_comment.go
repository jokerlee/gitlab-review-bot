package gitlab

import (
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"strings"
	"time"
)

func (c *Client) GetMergeRequestDiff(projectID int, mrID int) (string, error) {
	c.rl.Take()
	diffs, _, err := c.gitlab.MergeRequests.ListMergeRequestDiffs(
		projectID,
		mrID,
		&gitlab.ListMergeRequestDiffsOptions{
			Page:    1,
			PerPage: 20,
		})

	if err != nil {
		return "", errors.Wrap(err, "error get diffs of the merge request")
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

func (c *Client) AddCommentToMergeRequests(projectID int, mrID int, comment string) error {
	c.rl.Take()
	var now = time.Now()
	_, _, err := c.gitlab.Discussions.CreateMergeRequestDiscussion(
		projectID,
		mrID,
		&gitlab.CreateMergeRequestDiscussionOptions{
			Body:      &comment,
			CreatedAt: &now,
		})

	if err != nil {
		return errors.Wrap(err, "error add comment to merge request")
	}

	return nil
}
