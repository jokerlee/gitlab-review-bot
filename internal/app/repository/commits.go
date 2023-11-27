package repository

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jokerlee/gitlab-review-bot/internal/app/ds"
)

func (r *Repository) CommitByID(id string) (*ds.Commit, error) {
	ctx, cancel := context.WithTimeout(r.ctx, defaultTimeout)
	defer cancel()
	commit := &ds.Commit{}

	err := r.commits.FindOne(ctx, bson.D{{"id", id}}).Decode(&commit)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to find commit")
	}

	return commit, nil
}

func (r *Repository) CommitByProject(projectID int) ([]*ds.Commit, error) {
	ctx, cancel := context.WithTimeout(r.ctx, defaultTimeout)
	defer cancel()

	cursor, err := r.commits.Find(ctx, bson.D{{"project_id", projectID}})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find commits")
	}

	commits := make([]*ds.Commit, 0, 100)

	err = cursor.All(ctx, &commits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode commits")
	}

	return commits, nil
}

func (r *Repository) UpsertCommit(commit *ds.Commit) error {
	ctx, cancel := context.WithTimeout(r.ctx, defaultTimeout)
	defer cancel()

	opts := &options.UpdateOptions{}
	opts.SetUpsert(true)

	_, err := r.commits.UpdateOne(ctx,
		bson.D{{"id", commit.ID}},
		bson.D{{"$set", commit}},
		opts)
	if err != nil {
		return errors.Wrap(err, "failed to upsert commit")
	}

	return nil
}
