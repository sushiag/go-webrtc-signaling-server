package types

type Message struct {
	Type    string `json:"type"`
	Target  string `json:"target,omitempty"`
	Content string `json:"content"`
}
