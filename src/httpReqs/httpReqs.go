package httpReqs

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

func CrawlRequest(url string) (*http.Response, error) {
	// Make GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func RobotsRequest(baseUrl string) (string, error) {
	robotsUrl := fmt.Sprintf("%s/robots.txt", baseUrl)

	resp, err := http.Get(robotsUrl)
	if err != nil {
		return "", err
	}

	if resp.StatusCode > 399 {
		return "", errors.New("Unvalid response code")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
