package main

import (
	"crawler/src/common"
	"crawler/src/db"
	"crawler/src/httpReqs"
	"crawler/src/jsonData"
	"crawler/src/parsers"
	"io"
	"net/http"

	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jimsmart/grobotstxt"
	"github.com/puzpuzpuz/xsync/v3"

	"golang.org/x/net/html"
)

// Constants for crawler change by requirements
const numberOfWorkers int16 = 3
const respectRobots bool = true
const userAgent string = "grawler"
const dbName string = "web-crawler"
const descriptionLengthFromDocument int = 160
const titleLengthFromDocument int = 35
const hostCrawlDelay time.Duration = 400 * time.Millisecond
const documentExtension string = ".txt"

var allowedSchemes = map[string]bool{"http": true, "https": true}

// for specefic websites crawling
// var allowedHosts = map[string]bool{"en.wikipedia.org": true}

var seedList []string

var frontier *xsync.MPMCQueueOf[common.UrlData]

var crawledURLSMap *common.SafeBoolMap
var hostLastCrawledMap *common.SafeTimestampMap

func initializeMaps() {
	var err error
	seedList, err = jsonData.LoadSeedList()
	if err != nil {
		log.Fatal("error loading seed list:", err)
	}

	frontier = xsync.NewMPMCQueueOf[common.UrlData](100000000)
	crawledURLSMap = &common.SafeBoolMap{
		M: make(map[string]bool),
	}

	hostLastCrawledMap = &common.SafeTimestampMap{
		M: make(map[string]time.Time),
	}
}

func crawlAbleCheck(urlData common.UrlData, scheme string, host string) (bool, error) {
	if _, exists := allowedSchemes[scheme]; !exists {
		return false, fmt.Errorf("Scheme not allowed [%s]", scheme)
	}

	if crawledURLSMap.Get(urlData.URL) {
		return false, fmt.Errorf("Document has been crawled [%s]", urlData.URL)
	}

	// If you are confused this gets the last time a host has been crawled if a specefic delay hasent passed we cancel the request and add the url data to the frontier
	// to be crawled later
	if hostLastCrawledTimestamp, exists := hostLastCrawledMap.Get(host); exists && hostLastCrawledTimestamp.After(time.Now().Add(-hostCrawlDelay)) {
		frontier.Enqueue(urlData)
		return false, fmt.Errorf("Host delay still not completed")
	}

	pageExists, crawledTimestamp, err := db.CheckPageExistance(urlData.URL)
	if err != nil {
		log.Error("Error while checking page existance", "error", err)
	}

	if pageExists {
		oneAndHalfMonthsAgo := time.Now().AddDate(0, -1, -15)
		if crawledTimestamp.After(oneAndHalfMonthsAgo) {
			return false, fmt.Errorf("Document has been crawled recently [%s]", urlData.URL)
		}

		return true, nil
	}

	return false, nil
}

func checkRobots(url string, baseUrl string, host string) (bool, string, error) {
	robots, robotsRequestTimestamp, hostSaved, err := db.GetRobots(host)
	if err != nil {
		log.Error("error checking robots/host row existance", "error", err)
	}

	if hostSaved {
		oneAndHalfMonthsAgo := time.Now().AddDate(0, -1, -15)
		if robotsRequestTimestamp.Before(oneAndHalfMonthsAgo) {
			hostSaved = false

			robots, err = httpReqs.RobotsRequest(baseUrl)
			if err != nil {
				log.Error("error fetching for updating robots.txt", "host", baseUrl, "error", err)
			}
		}
		// log.Debug("Found robots info", "host", baseUrl)
	} else {
		robots, err = httpReqs.RobotsRequest(baseUrl)
		if err != nil {
			log.Error("error fetching robots.txt", "host", baseUrl, "error", err)
		}

		// log.Debug("Fetched robots.txt", "host", baseUrl)
	}

	if !grobotstxt.AgentAllowed(robots, userAgent, url) {
		log.Debug("", "URL", url)
		return false, "", fmt.Errorf("User agent not allowed by robots.txt [%s]", url)
	}

	if hostSaved {
		return true, "", nil
	}
	return false, robots, nil
}

func parseDocument(resp *http.Response, contentType string) (string, *html.Node, error) {
	defer resp.Body.Close()

	var pageText string
	var parsedHtml *html.Node
	var err error

	switch contentType {
	case "text/html":
		parsedHtml, err = html.Parse(resp.Body)
		if err != nil {
			return "", nil, fmt.Errorf("parse HTML failure: %v", err)
		}

		// Extract page-text
		pageText = parsers.ExtractPageText(parsedHtml, true)

	case "text/plain", "application/pdf":
		pageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", nil, fmt.Errorf("error reading bytes of reponse body: %v", err)
		}

		switch contentType {
		case "application/pdf":
			pageText, err = common.ReadPdfFromBytes(pageBytes)
			if err != nil {
				return "", nil, fmt.Errorf("error reading pdf from bytes: %v", err)
			}

		case "text/plain":
			pageText = string(pageBytes)
		}

	default:
		return "", nil, fmt.Errorf("Content-Type is not supported: %s", contentType)
	}

	pageText = strings.ReplaceAll(pageText, "\n", "")
	pageText = strings.ReplaceAll(pageText, "\r", "")
	pageText = strings.Trim(pageText, " ")

	return pageText, parsedHtml, nil
}

func extractMetaData(baseUrl string, host string, pageText string, contentType string, parsedHtml *html.Node) common.MetaData {
	var metaData common.MetaData
	var err error

	if contentType == "text/html" {
		subURLS := parsers.ExtractURLS(parsedHtml)
		// log.Debug("extracted URLS", "number of URLS", len(subURLS), "URL", urlData.URL)

		for _, url := range subURLS {
			if url != "" {
				if url[0] == '#' || url[0] == '?' {
					// subURLS[i] = ""
					// commented because we are not currently pushing suburls to the db
					continue
				}

				url, err = parsers.ConvertUrlToString(url)
				if err != nil {
					log.Error("URL to string failure", "error", err)
					continue
				}

				// CHECK IF THIS WORKS @TODO, im not sure but seems working needs more debugging
				if url[0] == '/' {
					url = fmt.Sprintf("%s%s", baseUrl, url)
					// subURLS[i] = url // This is so we update the list with the url, so its correct when pushing to the db
					// commented because we are not currently pushing suburls to the db
				}

				subUrlData := common.UrlData{
					URL:       url,
					ParentURL: url,
				}

				frontier.Enqueue(subUrlData)

				// Uncomment this if you want to see all extracted urls from a page .
				// log.Infof("Extracted: %v from %v .", url, urlData.URL)

			}
		}
		metaData = parsers.ExtractMetaData(parsedHtml)
		if metaData.Title == "" {
			metaData.Title = pageText[:min(titleLengthFromDocument, len(pageText))]
		}

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
	} else {
		metaData = common.MetaData{
			IconLink:    "",
			SiteName:    host,
			Title:       pageText[:min(titleLengthFromDocument, len(pageText))],
			Description: pageText[:min(descriptionLengthFromDocument, len(pageText))],
		}
	}

	return metaData
}

func crawlPage(urlData common.UrlData) {
	scheme, host, path, err := parsers.ExtractURLData(urlData.URL)
	if err != nil {
		log.Error("failed to extract base URL", "error", err)
		return
	}

	pageExists, err := crawlAbleCheck(urlData, scheme, host)
	if err != nil {
		log.Debug("Page not crawlable: %v", err)
	}

	baseUrl := fmt.Sprintf("%s://%s", scheme, host)

	hostSaved, robots, err := checkRobots(urlData.URL, baseUrl, host)
	if err != nil {
		log.Debug("Not allowed by robots: %v", err)
		return
	}

	log.Info("crawling", "URL", urlData.URL)

	resp, err := httpReqs.CrawlRequest(urlData.URL)
	if err != nil {
		log.Error("GET request Error", "URL", urlData.URL, "Error", err)
		return
	}

	if resp.StatusCode > 399 {
		log.Error("request error", "status-code", resp.StatusCode, "URL", urlData.URL)
		return
	}

	contentType := strings.Split(resp.Header.Get("content-type"), ";")[0]

	pageText, parsedHtml, err := parseDocument(resp, contentType)

	pageHash := common.HashSHA256(pageText)

	hashExists, err := db.CheckPageHash(pageHash)
	if err != nil {
		log.Error("failed to check page hash", "error", err)
		return
	}

	// Continue to the next url if page has an equivilant or the content hasen't been updated since last time crawled
	if hashExists {
		log.Warn("Hash already exists", "hash", pageHash, "current page url", urlData.URL)
		return
	}

	metaData := extractMetaData(baseUrl, host, pageText, contentType, parsedHtml)

	hostFolderPath := fmt.Sprintf("%s%s", common.DocumentsFolderName, strings.ReplaceAll(host, ":", "_"))

	if !hostSaved {
		hostShared := common.HostShared{
			Host:     host,
			Robots:   robots,
			IconLink: metaData.IconLink,
			SiteName: metaData.SiteName,
		}

		err := db.InsertHost(hostShared)
		if err != nil {
			log.Error("error inserting host shared data", "error", err, "host", host)
			return
		}

		// Create the host folder if not exists
		err = common.CreateFolder(hostFolderPath)
		if err != nil {
			log.Error("Error creating host folder %s", err)
			return
		}
	}

	// This basically checks if the first letter is a / for example /about and removes it
	if len(path) > 0 {
		if path[0] == '/' {
			path = strings.Replace(path, "/", "", 1)
		}
		path = strings.ReplaceAll(path, "/", "_")
	}
	// This replaces all / characters in a path with _ because when saving / causes problems since the system reads it as a new directory

	documentPath := fmt.Sprintf("%s/%s%s", hostFolderPath, path, documentExtension)

	page := &common.CrawledPage{
		URL:          urlData.URL,
		PageText:     pageText,
		ParentURL:    urlData.ParentURL,
		TimeCrawled:  time.Now(),
		PageHash:     pageHash,
		MetaData:     metaData,
		Host:         host,
		DocumentPath: documentPath,
	}

	if pageExists {
		err = db.UpdatePage(page)
		// Removing all page words before updating
		db.DeleteWords(urlData.URL)
	} else {
		err = db.InsertCrawledPage(page)
	}

	if err != nil {
		log.Error("error inserting/updating page:", "error", err, "URL", urlData.URL, "parent URL", urlData.ParentURL)
		return
	}

	re := regexp.MustCompile(`\b\w+\b`)
	words := re.FindAllString(pageText, -1)

	wordsFrequencies := make(map[string]int)

	for _, word := range words {
		word = strings.ToLower(word)
		wordsFrequencies[word]++
	}

	err = db.InsertWords(wordsFrequencies, urlData.URL)
	if err != nil {
		log.Error("error inserting words:", "error", err)
		return
	}

	hostLastCrawledMap.Set(host, time.Now())
	crawledURLSMap.Set(urlData.URL, true)

	return
}

func crawlWorker(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		start := time.Now()

		urlData := frontier.Dequeue()
		crawlPage(urlData)

		log.Debug("done crawling", "URL", urlData.URL, "time taken", time.Since(start))
	}
}

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Level:           log.DebugLevel,
	})
	log.SetDefault(logger)

	err := common.CreateFolder(common.DocumentsFolderName)
	if err != nil {
		log.Fatal("Failed to create documents folder", "Error", err)
	}

	err = db.InitPostgres("localhost", "5432", "postgres", "password", dbName)
	if err != nil {
		log.Fatal("failed to connect to PostgreSQL:", err)
	}
	defer db.ClosePostgres()

	initializeMaps()

	for _, url := range seedList {
		urlData := common.UrlData{
			URL:       url,
			ParentURL: nil,
		}
		frontier.Enqueue(urlData)
	}

	var wg sync.WaitGroup
	wg.Add(int(numberOfWorkers))

	for range numberOfWorkers {
		go crawlWorker(&wg)
	}

	wg.Wait()

}
