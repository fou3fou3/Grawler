package db

import (
	"crawler/src/common"
	"time"

	"github.com/couchbase/gocb/v2"
)

var cluster *gocb.Cluster
var documents *gocb.Bucket
var crawledDocuments *gocb.Collection

func InitCouchbase() error {
	var err error
	cluster, err = gocb.Connect("couchbase://localhost", gocb.ClusterOptions{
		Username: "Administrator",
		Password: "password",
	})
	if err != nil {
		return err
	}

	documents = cluster.Bucket("Documents")
	err = documents.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		return err
	}

	crawledDocuments = documents.Scope("CrawledDocuments").Collection("CrawledDocuments")
	return nil
}

func InsertDocument(document *common.Document) error {
	insertDocument := common.InsertDocument{
		ParentUrl: document.ParentUrl,
		Url:       document.Url,
		Response:  document.Response,
		Content:   document.Content,
		MetaData:  document.MetaData,
		Timestamp: time.Now(),
	}

	_, err := crawledDocuments.Insert(document.Url, insertDocument, &gocb.InsertOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	return nil
}
