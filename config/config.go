package config

import (
	"encoding/json"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/sdvdxl/go-tools/errors"

	"github.com/sdvdxl/logstash-http-push/log"

	"regexp"
	"strings"

	"github.com/sdvdxl/logstash-http-push/mail"
)

const (
	configFile = "cfg.json"
)

// Cfg 配置文件
var (
	cfg    *Config
	once   sync.Once
	inited = false
)

// Get 获取配置信息
func Get() *Config {
	if !inited {
		once.Do(load)
	}

	return cfg
}

// Config 配置文件
type Config struct {
	Negate         bool           `json:"negate"`  //是否取反
	MatchRegexText string         `json:"match"`   // 匹配的正则字符串
	MatchRegex     *regexp.Regexp `json:"-"`       // 匹配的正则字符串
	Address        string         `json:"address"` //web 服务地址 ":5678"
	LogLevel       string         `json:"logLevel"`
	MaxPerDay      uint64         `json:"maxPerDay"` //一天最大告警次数
	Filters        []*Filter      `json:"filters"`
	IsSendEmail    bool           `json:"sendEmail"` // true 发送
	TimeZone       int8           `json:"timeZone"`  //时区，如果时间有偏移则加上时区，否则设置为0即可
}

// Filter log 过滤
type Filter struct {
	lastMailIndex int
	Level         string          `json:"level"`
	Tags          []string        `json:"tags"`
	ToPerson      []string        `json:"toPerson"` // 邮件要发送的人
	Ignores       []string        `json:"ignores"`  // 忽略匹配的内容
	Ding          string          `json:"ding"`     // 钉钉 机器人token
	Mails         []mail.MailInfo `json:"mails"`
}

func (f *Filter) GetMail() mail.MailInfo {
	return f.Mails[f.lastMailIndex%len(f.Mails)]
}

func (f *Filter) GetNextMail() mail.MailInfo {
	f.lastMailIndex++

	return f.Mails[f.lastMailIndex%len(f.Mails)]
}

// Load 读取配置文件
func load() {
	bytes, err := ioutil.ReadFile(configFile)
	errors.Panic(err)

	var c Config
	errors.Panic(json.Unmarshal(bytes, &c))

	// 检查配置项目
	filters := c.Filters
	for i, v := range filters {
		sort.Strings(filters[i].Tags)

		if v.Level == "" {
			filters[i].Level = "ERROR"
		} else {
			filters[i].Level = strings.ToUpper(strings.TrimSpace(filters[i].Level))
		}
	}

	c.MatchRegex = regexp.MustCompile(c.MatchRegexText)

	configJSONText, _ := json.Marshal(c)
	log.Info("config init success, config:", string(configJSONText))
	cfg = &c
	inited = true
}
