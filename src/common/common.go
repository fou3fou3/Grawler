package common

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type RobotsItem struct {
	BaseUrl string `json:"base_url"`
	Robots  string `json:"robots"`
}

type UrlData struct {
	URL       string
	ParentURL interface{}
}

type MetaData struct {
	IconLink string

	SiteName    string
	Title       string
	Description string
}

// CrawledPage represents a web page in the database
type CrawledPage struct {
	Host     string
	MetaData MetaData

	ParentURL interface{}
	URL       string

	PageText string
	PageHash string

	TimeCrawled time.Time
}

type HostShared struct {
	Host     string
	Robots   string
	SiteName string
	IconLink string
}

// string safe map

type SafeStringMap struct {
	M map[string]string
	sync.Mutex
}

func (sm *SafeStringMap) Get(key string) (string, bool) {
	sm.Lock()
	defer sm.Unlock()
	val, ok := sm.M[key]
	return val, ok
}

func (sm *SafeStringMap) Set(key, value string) {
	sm.Lock()
	defer sm.Unlock()
	sm.M[key] = value
}

// bool safe map

type SafeBoolMap struct {
	M map[string]bool
	sync.Mutex
}

func (bm *SafeBoolMap) Get(key string) bool {
	bm.Lock()
	defer bm.Unlock()
	return bm.M[key]
}

func (bm *SafeBoolMap) Set(key string, value bool) {
	bm.Lock()
	defer bm.Unlock()
	bm.M[key] = value
}

// timestamp safe map

type SafeTimestampMap struct {
	M map[string]time.Time
	sync.Mutex
}

func (tm *SafeTimestampMap) Get(key string) (time.Time, bool) {
	tm.Lock()
	defer tm.Unlock()
	val, ok := tm.M[key]
	return val, ok
}

func (tm *SafeTimestampMap) Set(key string, value time.Time) {
	tm.Lock()
	defer tm.Unlock()
	tm.M[key] = value
}

// Queue type && functions

type Queue struct {
	Items []UrlData
}

func (q *Queue) Enqueue(data UrlData) {
	q.Items = append(q.Items, data)
}

func (q *Queue) Dequeue(numItems int16) {
	q.Items = q.Items[numItems:]
}

func (q *Queue) IsEmpty() bool {
	return len(q.Items) == 0
}

func RobotsListToMap(items []RobotsItem) map[string]string {
	result := make(map[string]string)
	for _, item := range items {
		result[item.BaseUrl] = item.Robots
	}
	return result
}

func RobotsMapToList(robotsMap map[string]string) []RobotsItem {
	items := make([]RobotsItem, 0, len(robotsMap))
	for baseUrl, robots := range robotsMap {
		items = append(items, RobotsItem{
			BaseUrl: baseUrl,
			Robots:  robots,
		})
	}
	return items
}

func HashSHA256(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}
