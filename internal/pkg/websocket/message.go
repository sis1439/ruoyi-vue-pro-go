package websocket

import (
	"encoding/json"
)

// Message matches Java JsonWebSocketMessage: content is serialized JSON inside a string.
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func NewMessage(msgType string, content interface{}) (*Message, error) {
	data, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return &Message{Type: msgType, Content: string(data)}, nil
}

// ToJSON 序列化为 JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage 解析 JSON 消息
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
