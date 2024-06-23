package parsers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
)

func ExtractURLS(n *html.Node) []string {
	var urls []string
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				urls = append(urls, attr.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		urls = append(urls, ExtractURLS(c)...)
	}
	return urls
}

func ExtractBaseURL(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

func ConvertUrlToString(encodedUrl string) (string, error) {
	decodedURL, err := url.QueryUnescape(encodedUrl)
	if err != nil {
		return "", err
	}

	return decodedURL, nil
}

func ExtractPageText(n *html.Node, trimSpace bool) string {
	switch n.Type {
	case html.TextNode:
		if trimSpace {
			return strings.TrimSpace(n.Data)
		}
		return n.Data
	case html.DocumentNode, html.ElementNode:
		// DocumentNode is the root, so we want to traverse its children
		// ElementNode is a regular tag like <p>, <div>, etc.

		// Skip unwanted tags
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "svg", "iframe":
				return ""
			}
		}

		var result strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			childText := ExtractPageText(c, trimSpace)
			if childText != "" {
				result.WriteString(childText)
				result.WriteRune(' ')
			}
		}
		return strings.TrimSpace(result.String())
	default:
		// Other node types like comments, doctypes, etc.
		return ""
	}
}

func IsUserAgentAllowed(robotsContent string, userAgent string, url string) (bool, error) {
	robots, err := robotstxt.FromBytes([]byte(robotsContent))
	if err != nil {
		return false, err
	}

	// Find the most specific group for the user agent
	group := robots.FindGroup(userAgent)
	if group == nil {
		group = robots.FindGroup("*")
	}

	// Check the Allow/Disallow rules
	allowed := group.Test(url)
	return allowed, nil
}

func ExtractTitle(n *html.Node) (string, error) {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return n.FirstChild.Data, nil
		}
		return "", nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title, err := ExtractTitle(c); title != "" || err != nil {
			return title, err
		}
	}
	return "", nil
}
