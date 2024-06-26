package db

import (
	"crawler/src/common"
	"database/sql"
	"fmt"

	// "time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// InitPostgres initializes the PostgreSQL connection
func InitPostgres(host, port, user, password, dbname string) error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	return db.Ping()
}

// ClosePostgres closes the PostgreSQL connection
func ClosePostgres() {
	db.Close()
}

// InsertCrawledPage inserts a new page into the crawled_pages table
func InsertCrawledPage(page *common.CrawledPage) error {
	_, err := db.Exec(`
        INSERT INTO crawled_pages (url, content, title, parent_url, timestamp, content_hash, host, icon_link, site_name, description)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, page.URL, page.PageText, page.MetaData.Title, page.ParentURL, page.TimeCrawled, page.PageHash, page.Host,
		page.MetaData.IconLink, page.MetaData.SiteName, page.MetaData.Description)
	return err
}

func InsertWords(wordsFrequencies map[string]int, parentUrl string, pageContentLength int) error {
	for word, freq := range wordsFrequencies {
		TF := float32(freq) / float32(pageContentLength)
		_, err := db.Exec(`
			INSERT INTO page_words (url, word, frequency)
			VALUES ($1, $2, $3)
		`, parentUrl, word, TF)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCrawledPage retrieves a page by its URL
func CheckPageHash(hash string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
        SELECT EXISTS (
            SELECT 1
            FROM crawled_pages 
            WHERE content_hash = $1
        )
    `, hash).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// **MIGHT NEED FOR LATER** // GetCrawledPage retrieves a page by its URL
// func GetCrawledPage(url string) (*common.CrawledPage, error) {
// 	var page common.CrawledPage
// 	err := db.QueryRow(`
//         SELECT page_url, page_content, created_at, parent_link
//         FROM crawled_pages WHERE page_url = $1
//     `, url).Scan(&page.URL, &page.PageText, &page.TimeCrawled, &page.ParentURL)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &page, nil
// }

// // UpdatePageText updates the page_content of a given URL
// func UpdatePageText(url, newText string) error {
// 	_, err := db.Exec(`
//         UPDATE crawled_pages
//         SET page_content = $1, created_at = $2
//         WHERE page_url = $3
//     `, newText, time.Now(), url)
// 	return err
// }

// // GetPagesByParent finds all pages with a given parent URL
// func GetPagesByParent(parentURL string) ([]string, error) {
// 	rows, err := db.Query(`
//         SELECT page_url FROM crawled_pages WHERE parent_link = $1
//     `, parentURL)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var childPages []string
// 	for rows.Next() {
// 		var url string
// 		if err := rows.Scan(&url); err != nil {
// 			return nil, err
// 		}
// 		childPages = append(childPages, url)
// 	}
// 	return childPages, rows.Err()
// }

// // GetRecentPages retrieves the most recently crawled pages
// func GetRecentPages(limit int) ([]*common.CrawledPage, error) {
// 	rows, err := db.Query(`
//         SELECT page_url, page_content, created_at, parent_link
//         FROM crawled_pages ORDER BY created_at DESC LIMIT $1
//     `, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var pages []*common.CrawledPage
// 	for rows.Next() {
// 		var page common.CrawledPage
// 		if err := rows.Scan(&page.URL, &page.PageText, &page.TimeCrawled, &page.ParentURL); err != nil {
// 			return nil, err
// 		}
// 		pages = append(pages, &page)
// 	}
// 	return pages, rows.Err()
// }

// // DeleteOldPages deletes pages not crawled since the given time
// func DeleteOldPages(olderThan time.Time) error {
// 	_, err := db.Exec(`
//         DELETE FROM crawled_pages WHERE created_at < $1
//     `, olderThan)
// 	return err
// }
