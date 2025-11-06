package types

type Message struct {
	Type      string      `json:"type"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}