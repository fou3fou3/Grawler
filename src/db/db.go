package db

import (
	"crawler/src/common"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

var cluster *gocb.Cluster

var documents *gocb.Bucket

var crawledDocuments *gocb.Collection
var robots *gocb.Collection
var words *gocb.Collection

var UpsertOptions = gocb.UpsertOptions{Timeout: 5 * time.Second}

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
	robots = documents.Scope("CrawledDocuments").Collection("Robots")
	words = documents.Scope("CrawledDocuments").Collection("Words")

	return nil
}

// Add upadting mechanism
func InsertDocument(document *common.Document) error {
	insertDocument := common.InsertDocument{
		ParentUrl: document.ParentUrl,
		Url:       document.Url,

		Response: document.Response,

		Content:  document.Content,
		MetaData: document.MetaData,

		Timestamp: time.Now(),
	}

	err := InsertWords(document.Url, document.Words)
	if err != nil {
		return err
	}

	_, err = crawledDocuments.Upsert(document.Url, insertDocument, &UpsertOptions)
	if err != nil {
		return err
	}
	return nil
}

func InsertWords(parentUrl string, Words map[string]int) error {
	for word, freq := range Words {
		_, err := words.Upsert(fmt.Sprintf("%s%s", parentUrl, word), common.Word{Word: word, Frequency: freq, ParentUrl: parentUrl}, &UpsertOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetRobots(host string) (*common.RobotsItem, bool, error) {
	var result common.RobotsItem
	result.Timestamp = time.Now() // This is so when we check if time was before a specefic date in agentAllowed function it doesn't give nil pointer err

	getResult, err := robots.Get(host, nil)
	if err != nil {
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	err = getResult.Content(&result)
	if err != nil {
		return nil, false, err
	}

	return &result, true, nil
}

func InsertRobots(robotsItem common.RobotsItem) error {
	_, err := robots.Upsert(robotsItem.Host, robotsItem, &UpsertOptions)
	if err != nil {
		return err
	}

	return nil
}
