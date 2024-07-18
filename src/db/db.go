package db

import (
	"context"
	"crawler/src/common"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var err error

func InitMongo() (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))

	return client, err
}

func InsertDocument(document *common.Document) error {
	// Make this a shared collection so we dont make a query each time
	collection := client.Database("crawler").Collection("documents")

	insertDocument := common.InsertDocument{
		ParentUrl: document.ParentUrl,
		Url:       document.Url,

		Response: document.Response,

		Content:  document.Content,
		MetaData: document.MetaData,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, insertDocument)
	if err != nil {
		return err
	}

	return nil
}
