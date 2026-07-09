package models

type Result struct {
	Caption     string
	ImageData   []byte
	SourceURL   string
	MessageType MessageType
	Text        string
}

type MessageType int

const (
	MessageTypeInstagramURL MessageType = iota
	MessageTypeOtherURL
	MessageTypeText
)
