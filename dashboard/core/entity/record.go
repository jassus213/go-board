package entity

type DashboardRecord struct {
	ID    string  `json:"id"`
	Rank  int64   `json:"rank"`
	Score float64 `json:"score"`
}
