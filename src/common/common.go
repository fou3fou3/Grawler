package common

import (
	"sync"
	"time"
)

const DocumentsFolderName string = "documents/"

type UrlComponents struct {
	Scheme string
	Host   string
	Path   string
}

type DocumentResponse struct {
	ContentType string
	StatusCode  int16
}

type MetaData struct {
	IconLink string

	SiteName    string
	Title       string
	Description string
}

type Document struct {
	ParentUrl     string
	Url           string
	BaseUrl       string
	UrlComponents UrlComponents

	Response DocumentResponse

	Content string
	Words   map[string]int

	MetaData  MetaData
	ChildUrls []string
}

type InsertDocument struct {
	ParentUrl string
	Url       string

	Response DocumentResponse

	Content  string
	MetaData MetaData

	Timestamp time.Time
}

type RobotsItem struct {
	Host   string `json:"base_url"`
	Robots string `json:"robots"`

	Timestamp time.Time
}

type Word struct {
	Word      string
	Frequency int
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
