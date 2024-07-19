package main

import (
	"crawler/src/common"
	"crawler/src/db"
	"crawler/src/parsers"
	"crawler/src/utils"
	"fmt"
	"time"

	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/jimsmart/grobotstxt"
	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/net/html"
)

const userAgent = "grawler"

// var seedList []string

var frontier *xsync.MPMCQueueOf[common.Document]

var crawledURLSMap *common.SafeBoolMap

// var hostLastCrawledMap *common.SafeTimestampMap

func initializeMaps() {
	// var err error
	// seedList, err = jsonData.LoadSeedList()
	// if err != nil {
	// 	log.Fatal("error loading seed list:", err)
	// }

	frontier = xsync.NewMPMCQueueOf[common.Document](100000)
	crawledURLSMap = &common.SafeBoolMap{
		M: make(map[string]bool),
	}

	// hostLastCrawledMap = &common.SafeTimestampMap{
	// 	M: make(map[string]time.Time),
	// }
}

func main() {
	utils.SetupLogger()
	initializeMaps()

	err := db.InitCouchbase()
	if err != nil {
		log.Fatal("Failed to connect to couchbase", "err", err)
	}

	baseDocument := common.Document{ParentUrl: "", Url: "https://en.wikipedia.org/wiki/Cosmic_microwave_background"}
	frontier.Enqueue(baseDocument)
	for {
		crawlDocument(frontier.Dequeue())
	}
}

func crawlDocument(document common.Document) {
	log.Info("Crawling", "url", document.Url)

	scheme, host, path, err := utils.ExtractUrlComponents(document.Url)
	if err != nil {
		log.Error("Extracting url components", "err", err)
		return
	}

	document.UrlComponents = common.UrlComponents{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	document.BaseUrl = fmt.Sprintf("%s://%s", scheme, host)

	if pageCrawled(document.Url) {
		log.Warn("Document has been crawled", "url", document.Url)
		return
	}

	if !urlAllowed(&document.UrlComponents) {
		log.Warn("Url not allowed", "url", document.Url)
		return
	}

	allowed, err := agentAllowed(&document)
	if err != nil {
		log.Error("Checking if user agent is allowed", "err", err)
		return
	}
	if !allowed {
		log.Warn("Not allowed by robots", "url", document.Url)
		return
	}

	resp, err := utils.HttpRequest("GET", document.Url, map[string]string{"User-Agent": userAgent})
	if err != nil {
		log.Error("Making crawl response", "err", err)
		return
	}

	err = handleCrawlResponse(resp, &document)
	if err != nil {
		log.Error("Handling crawl response", "err", err)
		return
	}

	allowed = documentAllowed(&document.Response)
	if !allowed {
		log.Warn("Document not allowed")
		return
	}

	err = parseDocument(&document)
	if err != nil {
		log.Error("Parsing document", "err", err)
		return
	}

	utils.PushChilds(frontier, &document)

	err = db.InsertDocument(&document)
	if err != nil {
		log.Error("Inserting document", "err", err)
		return
	}

	crawledURLSMap.Set(document.Url, true)

	log.Info("Done crawling", "url", document.Url)
}

func pageCrawled(url string) bool {
	if crawledURLSMap.Get(url) {
		return true
	}

	return false
}

func urlAllowed(urlComponents *common.UrlComponents) bool {
	var allowedSchemes = map[string]bool{"http": true, "https": true}
	var unallowedHosts = map[string]bool{}
	var unallowedPaths = map[string]bool{"/robots.txt": true}

	if _, exists := allowedSchemes[urlComponents.Scheme]; !exists {
		return false
	}
	if _, exists := unallowedHosts[urlComponents.Host]; exists {
		return false
	}
	if _, exists := unallowedPaths[urlComponents.Path]; exists {
		return false
	}

	return true
}

func agentAllowed(document *common.Document) (bool, error) {
	robotsItem, exists, err := db.GetRobots(document.UrlComponents.Host)
	if err != nil {
		return false, err
	}

	var robots string

	if !exists || robotsItem.Timestamp.Before(time.Now().AddDate(0, -1, -15)) {
		resp, err := utils.HttpRequest("GET", fmt.Sprintf("%s/robots.txt", document.BaseUrl), map[string]string{"User-Agent": userAgent})
		if err != nil {
			return false, err
		}

		defer resp.Body.Close()
		robotsBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}

		robots = string(robotsBytes)

		db.InsertRobots(common.RobotsItem{Host: document.UrlComponents.Host, Robots: robots, Timestamp: time.Now()})
	} else {
		robots = robotsItem.Robots
	}

	if !grobotstxt.AgentAllowed(robots, userAgent, document.Url) {
		return false, nil
	}

	return true, nil
}

func handleCrawlResponse(resp *http.Response, document *common.Document) error {
	contentType := strings.Split(strings.ToLower(resp.Header.Get("content-type")), ";")

	document.Response = common.DocumentResponse{
		ContentType: contentType[0],
		StatusCode:  int16(resp.StatusCode),
	}

	defer resp.Body.Close()
	documentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	document.Content = string(documentBytes)
	return nil
}

func documentAllowed(documentResponse *common.DocumentResponse) bool {
	var allowedContentTypes = map[string]bool{"text/html": true, "text/plain": true}

	if _, exists := allowedContentTypes[documentResponse.ContentType]; !exists {
		return false
	}

	return true
}

func parseDocument(document *common.Document) error {
	// @TODO add words & frequencies map to parsing
	// @TODO remove icon link from here that can be dealt with on host shared..
	switch document.Response.ContentType {
	case "text/html":
		document.Content = strings.ToValidUTF8(document.Content, "")

		parsedHtml, err := html.Parse(strings.NewReader(document.Content))
		if err != nil {
			return err
		}
		document.ChildUrls = parsers.HtmlUrls(parsedHtml)

		document.MetaData = parsers.HtmlMetaData(parsedHtml)
		pageText := parsers.HtmlText(parsedHtml, true)
		utils.FillTextDocEmptyMetaData(document, pageText)

		document.Content = pageText

	case "text/plain":
		document.MetaData = utils.DefaultMetaData()
		utils.FillTextDocEmptyMetaData(document, document.Content)
	}

	return nil
}
