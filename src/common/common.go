package common

type RobotsItem struct {
	BaseUrl string `json:"base_url"`
	Robots  string `json:"robots"`
}

type UrlData struct {
	URL       string
	ParentURL string
}

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
