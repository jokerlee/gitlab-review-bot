package gitlab

import (
	"github.com/pkg/errors"
	"github.com/spatecon/gitlab-review-bot/internal/app/service"
	"github.com/xanzy/go-gitlab"
	"time"
)

func (c *Client) GetMergeRequestDiff(projectID int, mrID int) (result []*service.Diff, err error) {
	c.rl.Take()
	diffs, _, err := c.gitlab.MergeRequests.ListMergeRequestDiffs(
		projectID,
		mrID,
		&gitlab.ListMergeRequestDiffsOptions{
			Page:    1,
			PerPage: 20,
		})

	if err != nil {
		err = errors.Wrap(err, "error get diffs of the merge request")
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
