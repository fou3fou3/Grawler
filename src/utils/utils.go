package utils

import (
	"bytes"
	"crawler/src/common"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/log"
	"github.com/ledongthuc/pdf"
	"github.com/puzpuzpuz/xsync/v3"
)

func HttpRequest(method string, url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}
	client.Timeout = 10 * time.Second

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for header, value := range headers {
		req.Header.Set(header, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 399 {
		return nil, fmt.Errorf("Error status code %v", resp.StatusCode)
	}

	return resp, nil
}

func ExtractUrlComponents(link string) (string, string, string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", "", "", err
	}
	return u.Scheme, u.Host, u.Path, nil
}

func convertUrlToString(encodedUrl string) (string, error) {
	decodedURL, err := url.QueryUnescape(encodedUrl)
	if err != nil {
		return "", err
	}

	return decodedURL, nil
}

// func RobotsListToMap(items []common.RobotsItem) map[string]string {
// 	result := make(map[string]string)
// 	for _, item := range items {
// 		result[item.BaseUrl] = item.Robots
// 	}
// 	return result
// }

// func RobotsMapToList(robotsMap map[string]string) []common.RobotsItem {
// 	items := make([]common.RobotsItem, 0, len(robotsMap))
// 	for baseUrl, robots := range robotsMap {
// 		items = append(items, common.RobotsItem{
// 			BaseUrl: baseUrl,
// 			Robots:  robots,
// 		})
// 	}
// 	return items
// }

func HashSHA256(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

func CreateFolder(folderName string) error {
	if _, err := os.Stat(folderName); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(folderName, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadPdfFromBytes(b []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}

	var content string
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			return "", err
		}
		content += text
	}
	return content, nil
}

func FillTextDocEmptyMetaData(document *common.Document, pageText string) {
	if document.MetaData.Title == "" {
		document.MetaData.Title = pageText[:min(60, len(pageText))]
	}

	if document.MetaData.Description == "" {
		document.MetaData.Description = pageText[:min(160, len(pageText))]
	}

	if document.MetaData.SiteName == "" {
		document.MetaData.SiteName = document.UrlComponents.Host
	}

	if document.MetaData.IconLink != "" {
		if document.MetaData.IconLink[0] == '/' {
			document.MetaData.IconLink = fmt.Sprintf("%s%s", document.BaseUrl, document.MetaData.IconLink)
		}
	}
}

func DefaultMetaData() common.MetaData {
	return common.MetaData{
		IconLink:    "",
		SiteName:    "",
		Title:       "",
		Description: "",
	}
}

func SetupLogger() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Level:           log.DebugLevel,
	})
	log.SetDefault(logger)
}

// Move to main function urlAllowed
func childUrlAllowed(url *string, baseUrl *string) bool {
	if *url == "" {
		return false
	}

	var err error

	*url, err = convertUrlToString(*url)
	if err != nil {
		log.Error("URL to string failure", "error", err)
		return false
	}

	if (*url)[0] == '#' || (*url)[0] == '?' {
		return false
	}

	if (*url)[0] == '/' {
		*url = fmt.Sprintf("%s%s", *baseUrl, *url)
	}

	return utf8.ValidString(*url)
}

func PushChilds(frontier *xsync.MPMCQueueOf[common.Document], document *common.Document) {
	for _, url := range document.ChildUrls {
		if childUrlAllowed(&url, &document.BaseUrl) {
			frontier.TryEnqueue(common.Document{
				ParentUrl: document.Url,
				Url:       url,
			})
		}
	}
}
