package main

import (
	"crawler/src/common"
	"crawler/src/db"
	"crawler/src/httpReqs"
	"crawler/src/jsonData"
	"crawler/src/parsers"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"

	"golang.org/x/net/html"
)

type SafeStringMap struct {
	m map[string]string
	sync.Mutex
}

type SafeBoolMap struct {
	m map[string]bool
	sync.Mutex
}

func (sm *SafeStringMap) Get(key string) (string, bool) {
	sm.Lock()
	defer sm.Unlock()
	val, ok := sm.m[key]
	return val, ok
}

func (sm *SafeStringMap) Set(key, value string) {
	sm.Lock()
	defer sm.Unlock()
	sm.m[key] = value
}

func (bm *SafeBoolMap) Get(key string) bool {
	bm.Lock()
	defer bm.Unlock()
	return bm.m[key]
}

func (bm *SafeBoolMap) Set(key string, value bool) {
	bm.Lock()
	defer bm.Unlock()
	bm.m[key] = value
}

func hashSHA256(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

const workers int16 = 5
const respectRobots bool = true
const userAgent string = "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0"
const dbName string = "search-engine"
const descriptionLengthFromDocument int = 160

func crawl(frontier *common.Queue, urlData common.UrlData, crawledURLSMap *SafeBoolMap, robotsMap *SafeStringMap,
	wg *sync.WaitGroup) {
	defer wg.Done()

	if crawledURLSMap.Get(urlData.URL) {
		log.Debug("Has been crawled", "URL", urlData.URL)
		return
	}

	scheme, host, err := parsers.ExtractBaseURL(urlData.URL)
	if err != nil {
		log.Error("Failed to extract base URL", "Error", err)
		return
	}
	baseUrl := fmt.Sprintf("%s://%s", scheme, host)

	if respectRobots {
		var robots string
		var exists bool

		robots, exists = robotsMap.Get(baseUrl)
		if exists {
			// log.Debug("Found robots info", "host", baseUrl)
		} else {
			robots, err := httpReqs.RobotsRequest(baseUrl)
			if err != nil {
				log.Error("Error fetching robots.txt", "host", baseUrl, "Error", err)
			}

			robotsMap.Set(baseUrl, robots)
			// log.Debug("Fetched robots.txt", "host", baseUrl)
		}

		robotsResult, err := parsers.IsUserAgentAllowed(robots, userAgent, urlData.URL)
		if err != nil {
			log.Error("Error checking user agent", "Error", err)
			return
		}

		if !robotsResult {
			log.Debug("Not allowed by robots", "URL", urlData.URL)
			return
		}
	}

	log.Info("Crawling", "URL", urlData.URL)

	resp, err := httpReqs.CrawlRequest(urlData.URL)
	if err != nil {
		log.Error("GET request Error", "URL", urlData.URL, "Error", err)
		return
	}

	defer resp.Body.Close()

	parsedHtml, err := html.Parse(resp.Body)
	if err != nil {
		log.Error("Parse HTML failure", "Error", err)
		return
	}

	// Extract page-text
	pageText := parsers.ExtractPageText(parsedHtml, true)
	pageHash := hashSHA256(pageText)

	pageExists, err := db.CheckPageHash(pageHash)
	if err != nil {
		log.Error("Failed to check page hash", "Error", err)
		return
	}

	if pageExists {
		log.Warn("Page already exists", "hash", pageHash, "current page url", urlData.URL)
		return
	}

	metaData := parsers.ExtractMetaData(parsedHtml)
	if metaData.Description == "" {
		metaData.Description = pageText[:min(descriptionLengthFromDocument, len(pageText))]
	}
	if metaData.SiteName == "" {
		metaData.SiteName = host
	}
	if metaData.IconLink != "" {
		if metaData.IconLink[0] == '/' {
			metaData.IconLink = fmt.Sprintf("%s%s", baseUrl, metaData.IconLink)
		}
	}

	// Extract links
	subURLS := parsers.ExtractURLS(parsedHtml)
	log.Debug("Extracted URLS", "Number of URLS", len(subURLS), "URL", urlData.URL)

	for _, url := range subURLS {
		if url != "" {
			if url[0] == '#' {
				// subURLS[i] = ""
				// commented because we are not currently pushing suburls to the db
				continue
			}

			url, err = parsers.ConvertUrlToString(url)
			if err != nil {
				log.Error("URL to string failure", "Error", err)
				return
			}

			if url[0] == '/' {
				url = fmt.Sprintf("%s%s", baseUrl, url)
				// subURLS[i] = url // This is so we update the list with the url, so its correct when pushing to the db
				// commented because we are not currently pushing suburls to the db
			}

			subUrlData := common.UrlData{
				URL:       url,
				ParentURL: urlData.URL,
			}

			frontier.Enqueue(subUrlData)

			// Uncomment this if you want to see all extracted urls from a page .
			// log.Infof("Extracted: %v from %v .", url, urlData.URL)

		}
	}

	page := &common.CrawledPage{
		URL:         urlData.URL,
		PageText:    pageText,
		ParentURL:   urlData.ParentURL,
		TimeCrawled: time.Now(),
		PageHash:    pageHash,
		MetaData:    metaData,
		Host:        host,
	}

	err = db.InsertCrawledPage(page)
	if err != nil {
		log.Error("Error inserting page:", "Error", err)
		return
	}

	re := regexp.MustCompile(`\b\w+\b`)
	words := re.FindAllString(pageText, -1)

	wordsFrequencies := make(map[string]int)

	for _, word := range words {
		word = strings.ToLower(word)
		wordsFrequencies[word]++
	}

	err = db.InsertWords(wordsFrequencies, urlData.URL, len(pageText))
	if err != nil {
		log.Error("Error inserting words:", "Error", err)
		return
	}

	crawledURLSMap.Set(urlData.URL, true)
	log.Info("Done Crawling", "URL", urlData.URL)
}

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Level:           log.DebugLevel,
	})
	log.SetDefault(logger)

	err := db.InitPostgres("localhost", "5432", "postgres", "password", dbName)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}
	defer db.ClosePostgres()

	seedList, err := jsonData.LoadSeedList()
	if err != nil {
		log.Fatal("Error loading seed list:", err)
		return
	}

	frontier := &common.Queue{}
	crawledURLSMap := &SafeBoolMap{
		m: make(map[string]bool),
	}

	robotsMap, err := jsonData.LoadRobotsMap()
	if err != nil {
		log.Fatal("Error loading the robots map:", err)
		return
	}

	safeRobotsMap := &SafeStringMap{
		m: robotsMap,
	}

	for _, url := range seedList {
		urlData := common.UrlData{
			URL:       url,
			ParentURL: nil,
		}
		frontier.Enqueue(urlData)
	}

	for !frontier.IsEmpty() {
		// start := time.Now()

		urlsData := frontier.Items[:min(int(workers), len(frontier.Items))]
		urlsDataLength := len(urlsData)
		frontier.Dequeue(int16(urlsDataLength))

		var wg sync.WaitGroup
		wg.Add(urlsDataLength)

		for _, urlData := range urlsData {
			go crawl(frontier, urlData, crawledURLSMap, safeRobotsMap, &wg)
		}

		wg.Wait()

		jsonData.DumpRobots(robotsMap)

		// elapsed := time.Since(start)

		// log.Warnf("Crawling %d URLS took %s", workers, elapsed)
	}

}
