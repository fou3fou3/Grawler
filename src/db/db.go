package db

import (
	"context"
	"crawler/src/common"
	"database/sql"
	"fmt"
	"time"

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
func InsertCrawledPage(crawledPage *common.CrawledPage) error {
	_, err := db.Exec(`
        INSERT INTO crawled_pages (url, content, title, parent_url, timestamp, content_hash, host, icon_link, site_name, description)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, crawledPage.URL, crawledPage.PageText, crawledPage.MetaData.Title, crawledPage.ParentURL, crawledPage.TimeCrawled, crawledPage.PageHash, crawledPage.Host,
		crawledPage.MetaData.IconLink, crawledPage.MetaData.SiteName, crawledPage.MetaData.Description)
	return err
}

// UpdatePageText updates the content of a given URL
func UpdatePage(crawledPage *common.CrawledPage) error {
	_, err := db.Exec(`
        UPDATE crawled_pages
        SET content = $1, title = $2, parent_url = $3, timestamp = $4, content_hash = $5, description = $6
        WHERE url = $7
    `, crawledPage.PageText, crawledPage.MetaData.Title, crawledPage.ParentURL, crawledPage.TimeCrawled, crawledPage.PageHash,
		crawledPage.MetaData.Description, crawledPage.URL)
	return err
}

func InsertWords(wordsFrequencies map[string]int, pageURL string) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	for word, freq := range wordsFrequencies {
		// (if you want word ) TF := float32(freq) / float32(pageContentLength)

		_, err := tx.Exec(`
			INSERT INTO page_words (url, word, frequency)
			VALUES ($1, $2, $3)
		`, pageURL, word, freq)
		if err != nil {
			return err
		}

	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func DeleteWords(pageURL string) error {
	_, err := db.Exec("DELETE FROM page_words WHERE url = $1", pageURL)
	return err
}

// Check if a page has been crawled in db
func CheckPageExistance(pageURL string) (bool, time.Time, error) {
	var timestamp time.Time

	err := db.QueryRow(`
        SELECT timestamp
        FROM crawled_pages 
        WHERE url = $1
    `, pageURL).Scan(&timestamp)

	if err == sql.ErrNoRows {
		return false, time.Time{}, nil
	} else if err != nil {
		return false, time.Time{}, err
	}

	return true, timestamp, nil
}

// Checks if page's hash exists
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
