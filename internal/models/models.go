package models

type FetchRequest struct {
	URL            string   `json:"url"`
	Proxies        []string `json:"proxies"`
	Mode           string   `json:"mode"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type FetchResponse struct {
	Status       string `json:"status"`
	WinningProxy string `json:"winning_proxy,omitempty"`
	Content      string `json:"content,omitempty"`
	TimeTakenMs  int64  `json:"time_taken_ms,omitempty"`
	Message      string `json:"message,omitempty"`
}

type RaceResult struct {
	Proxy   string
	Content string
}
