package main

import (
	"crawler/src/common"
	"crawler/src/db"
	"crawler/src/httpReqs"
	"crawler/src/jsonData"
	"crawler/src/parsers"

	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jimsmart/grobotstxt"

	"golang.org/x/net/html"
)

// Constants for crawler change by requirements
const workers int16 = 3
const respectRobots bool = true
const userAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
const dbName string = "web-crawler"
const descriptionLengthFromDocument int = 160
const hostCrawlDelay time.Duration = 400 * time.Millisecond

// for specefic websites crawling
// var allowedHosts = map[string]bool{"en.wikipedia.org": true}

func crawl(frontier *common.Queue, urlData common.UrlData, crawledURLSMap *common.SafeBoolMap, hostLastCrawledMap *common.SafeTimestampMap,
	wg *sync.WaitGroup) {
	defer wg.Done()

	if crawledURLSMap.Get(urlData.URL) {
		log.Debug("Has been crawled", "URL", urlData.URL)
		return
	}

	pageExists, crawledTimestamp, err := db.CheckPageExistance(urlData.URL)
	if err != nil {
		log.Error("Error while checking page existance", "error", err)
	}

	if pageExists {
		oneAndHalfMonthsAgo := time.Now().AddDate(0, -1, -15)
		if crawledTimestamp.After(oneAndHalfMonthsAgo) {
			// log.Debug("Page has been crawled recently", "URL", urlData.URL)
			return
		}
	}

	scheme, host, err := parsers.ExtractURLData(urlData.URL)
	if err != nil {
		log.Error("Failed to extract base URL", "Error", err)
		return
	}

	// If you are confused this gets the last time a host has been crawled if a specefic delay hasent passed we cancel the request and add the url data to the frontier
	// to be crawled later
	if hostLastCrawledTimestamp, exists := hostLastCrawledMap.Get(host); exists && hostLastCrawledTimestamp.After(time.Now().Add(-hostCrawlDelay)) {
		// log.Debug("host delay still not completed")
		frontier.Enqueue(urlData)
		return
	}

	// for specefic websites crawling
	// if _, exists := allowedHosts[host]; !exists {
	// 	return
	// }

	baseUrl := fmt.Sprintf("%s://%s", scheme, host)

	//  _____   ____  ____   ____ _______ _____
	// |  __ \ / __ \|  _ \ / __ \__   __/ ____|
	// | |__) | |  | | |_) | |  | | | | | (___
	// |  _  /| |  | |  _ <| |  | | | |  \___ \
	// | | \ \| |__| | |_) | |__| | | |  ____) |
	// |_|  \_\\____/|____/ \____/  |_| |_____/

	robots, hostSaved, err := db.GetRobots(host)
	if err != nil {
		log.Error("Error checking robots/host row existance", "Error", err)
	}

	if hostSaved {
		// log.Debug("Found robots info", "host", baseUrl)
	} else {
		robots, err = httpReqs.RobotsRequest(baseUrl)
		if err != nil {
			log.Error("Error fetching robots.txt", "host", baseUrl, "Error", err)
		}

		// log.Debug("Fetched robots.txt", "host", baseUrl)
	}

	if !grobotstxt.AgentAllowed(robots, userAgent, urlData.URL) {
		log.Debug("User agent not allowed by robots.txt", "URL", urlData.URL)
		return
	}

	// 	 _____ _____       __          ___      _____ _   _  _____
	//  / ____|  __ \     /\ \        / / |    |_   _| \ | |/ ____|
	// | |    | |__) |   /  \ \  /\  / /| |      | | |  \| | |  __
	// | |    |  _  /   / /\ \ \/  \/ / | |      | | | . ` | | |_ |
	// | |____| | \ \  / ____ \  /\  /  | |____ _| |_| |\  | |__| |
	//  \_____|_|  \_\/_/    \_\/  \/   |______|_____|_| \_|\_____|

	log.Info("Crawling", "URL", urlData.URL)

	resp, err := httpReqs.CrawlRequest(urlData.URL)
	if err != nil {
		log.Error("GET request Error", "URL", urlData.URL, "Error", err)
		return
	}

	if resp.StatusCode > 399 {
		log.Error("Request Error", "status-code", resp.StatusCode, "URL", urlData.URL)
		return
	}

	defer resp.Body.Close()

	//  _____        _____   _____ _____ _   _  _____
	// |  __ \ /\   |  __ \ / ____|_   _| \ | |/ ____|
	// | |__) /  \  | |__) | (___   | | |  \| | |  __
	// |  ___/ /\ \ |  _  / \___ \  | | | . ` | | |_ |
	// | |  / ____ \| | \ \ ____) |_| |_| |\  | |__| |
	// |_| /_/    \_\_|  \_\_____/|_____|_| \_|\_____|

	parsedHtml, err := html.Parse(resp.Body)
	if err != nil {
		log.Error("Parse HTML failure", "Error", err)
		return
	}

	// Extract page-text
	pageText := parsers.ExtractPageText(parsedHtml, true)
	pageHash := common.HashSHA256(pageText)

	hashExists, err := db.CheckPageHash(pageHash)
	if err != nil {
		log.Error("Failed to check page hash", "Error", err)
		return
	}

	// Return if page has an equivilant or the content hasen't been updated since last time crawled
	if hashExists {
		// log.Warn("Hash already exists", "hash", pageHash, "current page url", urlData.URL)
		return
	}

	//  ______ _____   ____  _   _ _______ _____ ______ _____    _____  _    _  _____ _    _ _____ _   _  _____
	// |  ____|  __ \ / __ \| \ | |__   __|_   _|  ____|  __ \  |  __ \| |  | |/ ____| |  | |_   _| \ | |/ ____|
	// | |__  | |__) | |  | |  \| |  | |    | | | |__  | |__) | | |__) | |  | | (___ | |__| | | | |  \| | |  __
	// |  __| |  _  /| |  | | . ` |  | |    | | |  __| |  _  /  |  ___/| |  | |\___ \|  __  | | | | . ` | | |_ |
	// | |    | | \ \| |__| | |\  |  | |   _| |_| |____| | \ \  | |    | |__| |____) | |  | |_| |_| |\  | |__| |
	// |_|    |_|  \_\\____/|_| \_|  |_|  |_____|______|_|  \_\ |_|     \____/|_____/|_|  |_|_____|_| \_|\_____|

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

			// CHECK IF THIS WORKS @TODO
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

	//  __  __ ______ _______       _____       _______
	// |  \/  |  ____|__   __|/\   |  __ \   /\|__   __|/\
	// | \  / | |__     | |  /  \  | |  | | /  \  | |  /  \
	// | |\/| |  __|    | | / /\ \ | |  | |/ /\ \ | | / /\ \
	// | |  | | |____   | |/ ____ \| |__| / ____ \| |/ ____ \
	// |_|  |_|______|  |_/_/    \_\_____/_/    \_\_/_/    \_\

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

	//  _    _  ____   _____ _______    _____ _    _          _____  ______ _____    _____ _   _  _____ ______ _____ _______
	// | |  | |/ __ \ / ____|__   __|  / ____| |  | |   /\   |  __ \|  ____|  __ \  |_   _| \ | |/ ____|  ____|  __ \__   __|
	// | |__| | |  | | (___    | |    | (___ | |__| |  /  \  | |__) | |__  | |  | |   | | |  \| | (___ | |__  | |__) | | |
	// |  __  | |  | |\___ \   | |     \___ \|  __  | / /\ \ |  _  /|  __| | |  | |   | | | . ` |\___ \|  __| |  _  /  | |
	// | |  | | |__| |____) |  | |     ____) | |  | |/ ____ \| | \ \| |____| |__| |  _| |_| |\  |____) | |____| | \ \  | |
	// |_|  |_|\____/|_____/   |_|    |_____/|_|  |_/_/    \_\_|  \_\______|_____/  |_____|_| \_|_____/|______|_|  \_\ |_|

	if !hostSaved {
		hostShared := common.HostShared{
			Host:     host,
			Robots:   robots,
			IconLink: metaData.IconLink,
			SiteName: metaData.SiteName,
		}

		err := db.InsertHost(hostShared)
		if err != nil {
			log.Error("Error inserting host shared data", "Error", err, "Host", host)
			return
		}
	}

	// 	 _____ _____       __          ___      ______ _____    _____        _____ ______   _____ _   _  _____ ______ _____ _______
	//  / ____|  __ \     /\ \        / / |    |  ____|  __ \  |  __ \ /\   / ____|  ____| |_   _| \ | |/ ____|  ____|  __ \__   __|
	// | |    | |__) |   /  \ \  /\  / /| |    | |__  | |  | | | |__) /  \ | |  __| |__      | | |  \| | (___ | |__  | |__) | | |
	// | |    |  _  /   / /\ \ \/  \/ / | |    |  __| | |  | | |  ___/ /\ \| | |_ |  __|     | | | . ` |\___ \|  __| |  _  /  | |
	// | |____| | \ \  / ____ \  /\  /  | |____| |____| |__| | | |  / ____ \ |__| | |____   _| |_| |\  |____) | |____| | \ \  | |
	//  \_____|_|  \_\/_/    \_\/  \/   |______|______|_____/  |_| /_/    \_\_____|______| |_____|_| \_|_____/|______|_|  \_\ |_|

	page := &common.CrawledPage{
		URL:         urlData.URL,
		PageText:    pageText,
		ParentURL:   urlData.ParentURL,
		TimeCrawled: time.Now(),
		PageHash:    pageHash,
		MetaData:    metaData,
		Host:        host,
	}

	if pageExists {
		err = db.UpdatePage(page)
		// Removing all page words before updating
		db.DeleteWords(urlData.URL)
	} else {
		err = db.InsertCrawledPage(page)
	}

	if err != nil {
		log.Error("Error inserting/updating page:", "Error", err, "URL", urlData.URL, "Parent URL", urlData.ParentURL)
		return
	}

	// __          ______  _____  _____   _____   _____ _   _  _____ ______ _____ _______
	// \ \        / / __ \|  __ \|  __ \ / ____| |_   _| \ | |/ ____|  ____|  __ \__   __|
	//  \ \  /\  / / |  | | |__) | |  | | (___     | | |  \| | (___ | |__  | |__) | | |
	//   \ \/  \/ /| |  | |  _  /| |  | |\___ \    | | | . ` |\___ \|  __| |  _  /  | |
	//    \  /\  / | |__| | | \ \| |__| |____) |  _| |_| |\  |____) | |____| | \ \  | |
	// 	   \/  \/   \____/|_|  \_\_____/|_____/  |_____|_| \_|_____/|______|_|  \_\ |_|

	re := regexp.MustCompile(`\b\w+\b`)
	words := re.FindAllString(pageText, -1)

	wordsFrequencies := make(map[string]int)

	for _, word := range words {
		word = strings.ToLower(word)
		wordsFrequencies[word]++
	}

	err = db.InsertWords(wordsFrequencies, urlData.URL)
	if err != nil {
		log.Error("Error inserting words:", "Error", err)
		return
	}

	hostLastCrawledMap.Set(host, time.Now())
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

	}

	frontier := &common.Queue{}
	crawledURLSMap := &common.SafeBoolMap{
		M: make(map[string]bool),
	}

	hostLastCrawledMap := &common.SafeTimestampMap{
		M: make(map[string]time.Time),
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
			go crawl(frontier, urlData, crawledURLSMap, hostLastCrawledMap, &wg)
		}

		wg.Wait()

		// elapsed := time.Since(start)

		// log.Warnf("Crawling %d URLS took %s", workers, elapsed)
	}

}
