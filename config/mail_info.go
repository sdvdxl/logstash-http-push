package config

import (
	"sync"
	"time"
)

// MailInfo 邮件信息
type MailInfo struct {
	Lock         sync.Mutex
	Duration     int          `json:"duration" mapstructure:"duration"` //秒如果大于0，则每隔 duration 秒批量发送一封邮件，否则立刻发送
	Ticker       *time.Ticker `json:"-" mapstructure:"-"`
	MailMessages []string     `json:"-" mapstructure:"-"`
	ToPersons    []string     `json:"toPersons" mapstructure:"toPersons"`
	Name         string       `json:"-" mapstructure:"-"`
	Enable       bool         `json:"enable" mapstructure:"enable"`
	Senders      []MailSender `json:"senders" mapstructure:"senders"`
}

type MailSender struct {
	SMTP     string `json:"smtp" mapstructure:"smtp"`
	Port     int    `json:"port" mapstructure:"port"`
	Sender   string `json:"sender" mapstructure:"sender"`
	Password string `json:"password" mapstructure:"password"`
}
