package config

import (
	"sync"
	"time"
)

// MailInfo 邮件信息
type MailInfo struct {
	Lock         sync.Mutex
	Duration     int          `json:"duration"` //秒如果大于0，则每隔 duration 秒批量发送一封邮件，否则立刻发送
	Ticker       *time.Ticker `json:"-"`
	MailMessages []string     `json:"-"`
	ToPersons    []string     `json:"toPersons"`
	Name         string       `json:"-"`
	Enable       bool         `json:"enable"`
	Senders      []MailSender `json:"senders"`
}

type MailSender struct {
	SMTP     string `json:"smtp"`
	Port     int    `json:"port"`
	Sender   string `json:"sender"`
	Password string `json:"password"`
}
