package config

import (
	"encoding/json"
	"io/ioutil"
	"sort"

	"github.com/sdvdxl/logstash-http-push/log"

	"strings"

	"regexp"

	"github.com/fsnotify/fsnotify"
)

const (
	configFile = "cfg.json"
)

// Cfg 配置文件
var cfg *Config

// Get 获取配置信息
func Get() *Config {
	return cfg
}

// Config 配置文件
type Config struct {
	Negate      bool       `json:"negate"`  //是否取反
	Match       string     `json:"match"`   // 匹配的正则字符串
	Address     string     `json:"address"` //web 服务地址 ":5678"
	LogLevel    string     `json:"logLevel"`
	MaxPerDay   uint64     `json:"maxPerDay"` //一天最大告警次数
	Filters     []Filter   `json:"filters"`
	Mails       []MailInfo `json:"mails"`
	IsSendEmail bool       `json:"sendEmail"` // true 发送
	TimeZone    int8       `json:"timeZone"`  //时区，如果时间有偏移则加上时区，否则设置为0即可
}

// MailInfo 邮件信息
type MailInfo struct {
	SMTP     string `json:"smtp"`
	Port     int    `json:"port"`
	Sender   string `json:"sender"`
	Password string `json:"password"`
}

// Filter log 过滤
type Filter struct {
	Level    string   `json:"level"`
	Tags     []string `json:"tags"`
	ToPerson []string `json:"toPerson"` // 邮件要发送的人
	Ignores  []string `json:"ignores"`  // 忽略匹配的内容
	Ding     string   `json:"ding"`     // 钉钉 机器人token
}

// Load 读取配置文件
func Load() error {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	var c Config
	if err := json.Unmarshal(bytes, &c); err != nil {
		return err
	}

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

	if _, err = regexp.Compile(c.Match); err != nil {
		return err
	}

	configJSONText, _ := json.Marshal(c)
	log.Info("config init success, config:", string(configJSONText))
	cfg = &c
	return nil
}

// WatchConfigFileStatus 配置文件监控,可以实现热更新
func (cfg Config) WatchConfigFileStatus() chan bool {
	statusChanged := make(chan bool, 1)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				log.Debug(event)
				if event.Name == configFile && event.Op != fsnotify.Chmod { //&& (event.Op == fsnotify.Chmod || event.Op == fsnotify.Rename || event.Op == fsnotify.Write || event.Op == fsnotify.Create)
					log.Info("modified config file", event.Name, "will reaload config")
					if err := Load(); err != nil {
						log.Warn("ERROR: config has error, will use old config", err)
					} else {
						statusChanged <- true
						log.Info("config reload success")
					}

				}
			case err := <-watcher.Errors:
				log.Fatal(err)
			}
		}
	}()

	err = watcher.Add("cfg.json")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("watching config file...")
	statusChanged <- true
	return statusChanged
}
