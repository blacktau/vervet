package connections

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServerInformation struct {
	Version string `json:"version"`
}

type BuildInfo struct {
	Version    string `bson:"version"`
	GitVersion string `bson:"gitVersion"`
	Ok         int    `bson:"ok"`
}

func getBuildInfo(client *mongo.Client, ctx context.Context) (*BuildInfo, error) {
	var result BuildInfo
	err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve mongo version: %w", err)
	}

	return &result, nil
}
