package parsers

import (
	"crawler/src/common"
	"net/url"
	"strings"

	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
)

func ExtractBaseURL(link string) (string, string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", "", err
	}
	return u.Scheme, u.Host, nil
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

func ExtractMetaData(n *html.Node) common.MetaData {
	var metaData common.MetaData

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "link":
				var rel, href string
				for _, a := range n.Attr {
					switch a.Key {
					case "rel":
						rel = a.Val
					case "href":
						href = a.Val

						if (rel == "icon" || rel == "icon shortcut") && metaData.IconLink == "" {
							metaData.IconLink = href
						}
					}
				}

			case "title":
				if n.FirstChild != nil && metaData.Title == "" {
					metaData.Title = n.FirstChild.Data
				}
			case "meta":
				var name, property, content string
				for _, a := range n.Attr {
					switch a.Key {
					case "name":
						name = a.Val
					case "property":
						property = a.Val
					case "content":
						content = a.Val
					}
				}
				if (name == "description" || property == "og:description") && metaData.Description == "" {
					metaData.Description = content
				}
				if property == "og:site_name" && metaData.SiteName == "" {
					metaData.SiteName = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(n)
	return metaData
}

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
