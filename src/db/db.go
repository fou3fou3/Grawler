package db

import (
	"time"

	"github.com/gocql/gocql"
)

// Some functions in this file are not used **BUT** theire for future development !

var session *gocql.Session

// InitCassandra initializes the Cassandra connection
func InitCassandra(hosts []string, keyspace string) error {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 5 * time.Second

	var err error
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}
	return nil
}

// CloseCassandra closes the Cassandra session
func CloseCassandra() {
	session.Close()
}

// CrawledPage represents a web page in the database
type CrawledPage struct {
	URL         string
	PageText    string
	ChildURLs   []string
	TimeCrawled time.Time
	ParentURL   string
}

// InsertCrawledPage inserts a new page into the crawled_pages table
func InsertCrawledPage(page *CrawledPage) error {
	return session.Query(`
        INSERT INTO crawled_pages (url, page_text, child_urls, time_crawled, parent_url)
        VALUES (?, ?, ?, toTimestamp(now()), ?)
    `, page.URL, page.PageText, page.ChildURLs, page.ParentURL).Exec()
}

// GetCrawledPage retrieves a page by its URL
func GetCrawledPage(url string) (*CrawledPage, error) {
	var page CrawledPage
	err := session.Query(`
        SELECT url, page_text, child_urls, time_crawled, parent_url
        FROM crawled_pages WHERE url = ?
    `, url).Scan(&page.URL, &page.PageText, &page.ChildURLs, &page.TimeCrawled, &page.ParentURL)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// UpdatePageText updates the page_text of a given URL
func UpdatePageText(url, newText string) error {
	return session.Query(`
        UPDATE crawled_pages
        SET page_text = ?, time_crawled = toTimestamp(now())
        WHERE url = ?
    `, newText, url).Exec()
}

// AddChildURL adds a new URL to the child_urls set
func AddChildURL(url, childURL string) error {
	return session.Query(`
        UPDATE crawled_pages
        SET child_urls = child_urls + {?}
        WHERE url = ?
    `, childURL, url).Exec()
}

// GetPagesByParent finds all pages with a given parent URL
func GetPagesByParent(parentURL string) ([]string, error) {
	var childPages []string
	iter := session.Query(`
        SELECT url FROM crawled_pages WHERE parent_url = ?
    `, parentURL).Iter()

	var url string
	for iter.Scan(&url) {
		childPages = append(childPages, url)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return childPages, nil
}

// GetRecentPages retrieves the most recently crawled pages
func GetRecentPages(limit int) ([]*CrawledPage, error) {
	var pages []*CrawledPage
	iter := session.Query(`
        SELECT url, page_text, child_urls, time_crawled, parent_url
        FROM crawled_pages ORDER BY time_crawled DESC LIMIT ?
    `, limit).Iter()

	var page CrawledPage
	for iter.Scan(&page.URL, &page.PageText, &page.ChildURLs, &page.TimeCrawled, &page.ParentURL) {
		pages = append(pages, &page)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return pages, nil
}

// DeleteOldPages deletes pages not crawled since the given time
func DeleteOldPages(olderThan time.Time) error {
	return session.Query(`
        DELETE FROM crawled_pages WHERE time_crawled < ?
    `, olderThan).Exec()
}
