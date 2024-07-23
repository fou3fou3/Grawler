package parsers

import (
	"regexp"
	"strings"
)

func ProcessText(text string) string {
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.Trim(text, " ")

	return text
}

func TextWordsFreq(text string) map[string]int {

	re := regexp.MustCompile(`\b\w+\b`)
	words := re.FindAllString(text, -1)

	wordsFrequencies := make(map[string]int)

	for _, word := range words {
		word = strings.ToLower(word)
		wordsFrequencies[word]++
	}

	return wordsFrequencies
}
