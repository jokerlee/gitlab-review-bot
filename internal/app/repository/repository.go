package repository

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultTimeout = 5 * time.Second

type Repository struct {
	ctx context.Context

	conn *mongo.Client

	teams          *mongo.Collection
	projects       *mongo.Collection
	mergeRequests  *mongo.Collection
	commits        *mongo.Collection
	policyMetadata *mongo.Collection
}

func New(rootCtx context.Context, conn *mongo.Client, databaseName string) (*Repository, error) {
	database := conn.Database(databaseName)

	r := &Repository{
		ctx:            rootCtx,
		conn:           conn,
		teams:          database.Collection("teams"),
		projects:       database.Collection("projects"),
		mergeRequests:  database.Collection("merge_requests"),
		commits:        database.Collection("commits"),
		policyMetadata: database.Collection("policy_metadata"),
	}

	err := r.createIndexes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create indexes")
	}

	return r, nil
}

func (r *Repository) createIndexes() error {
	ctx, cancel := context.WithTimeout(r.ctx, defaultTimeout)
	defer cancel()

	_, err := r.mergeRequests.Indexes().CreateMany(ctx,
		[]mongo.IndexModel{
			{
				Keys:    bson.D{{"iid", 1}, {"project_id", 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys:    bson.D{{"id", 1}},
				Options: options.Index().SetUnique(true),
			},
		})
	if err != nil {
		return errors.Wrap(err, "failed to create merge_requests indexes")
	}

	_, err = r.commits.Indexes().CreateMany(ctx,
		[]mongo.IndexModel{
			{
				Keys:    bson.D{{"project_id", 1}},
				Options: options.Index(),
			},
			{
				Keys:    bson.D{{"id", 1}},
				Options: options.Index().SetUnique(true),
			},
		})
	if err != nil {
		return errors.Wrap(err, "failed to create commit indexes")
	}

	_, err = r.policyMetadata.Indexes().CreateMany(context.Background(),
		[]mongo.IndexModel{
			{
				Keys:    bson.D{{"mr_id", 1}, {"team_id", 1}, {"policy_name", 1}},
				Options: options.Index().SetUnique(true),
			},
		})
	if err != nil {
		return errors.Wrap(err, "failed to create policy_metadata indexes")
	}

	return nil
}
