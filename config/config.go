package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/log-dog/logstash-http-push/log"

	"strings"

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
	Address   string   `json:"address"` //web 服务地址 ":5678"
	LogLevel  string   `json:"logLevel"`
	MaxPerDay uint8    `json:"maxPerDay"` //一天最大告警次数
	Filters   []Filter `json:"filters"`
	Mail      MailInfo `json:"mail"`
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
	Level     string   `json:"level"`
	Tags      []string `json:"tags"`
	ToPersion []string `json:"toPersion"` // 邮件要发送的人
}

// Load 读取配置文件
func Load() error {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return err
	}

	// 检查配置项目
	filters := cfg.Filters
	for i, v := range filters {
		if v.Level == "" {
			filters[i].Level = "ERROR"
		} else {
			filters[i].Level = strings.ToUpper(strings.TrimSpace(filters[i].Level))
		}
	}

	configJSONText, _ := json.Marshal(cfg)
	log.Info("config init success, config:", string(configJSONText))
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

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("watching config file...")
	statusChanged <- true
	return statusChanged
}
