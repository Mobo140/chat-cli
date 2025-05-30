package root

import "time"

type Session struct {
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	Username     string    `json:"username"`
	Messages     []Message `json:"messages"`
}

type Message struct {
	ChatID   string    `json:"chat_id"`
	Username string    `json:"username"`
	Text     string    `json:"text"`
	Time     time.Time `json:"time"`
}
