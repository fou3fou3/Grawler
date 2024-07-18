package parsers

import (
	"crawler/src/common"
	"strings"

	"golang.org/x/net/html"
)

func HtmlMetaData(n *html.Node) common.MetaData {
	var metaData common.MetaData
	metaData.Description = ""
	metaData.IconLink = ""
	metaData.SiteName = ""
	metaData.Title = ""

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

						if (rel == "icon" || rel == "icon shortcut" || rel == "shortcut icon") && metaData.IconLink == "" {
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

func HtmlUrls(n *html.Node) []string {
	var urls []string
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				urls = append(urls, attr.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		urls = append(urls, HtmlUrls(c)...)
	}
	return urls
}

func HtmlText(n *html.Node, trimSpace bool) string {
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
			childText := HtmlText(c, trimSpace)
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
