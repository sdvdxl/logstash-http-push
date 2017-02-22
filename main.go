package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/log-dog/logstash-http-push/log"

	"strings"

	"fmt"

	"encoding/base64"

	"github.com/log-dog/logstash-http-push/config"
	"github.com/log-dog/logstash-http-push/logstash"
	"github.com/log-dog/logstash-http-push/mail"
)

var (
	tagsMap      map[string]map[string]bool
	alarmInfoMap map[string]uint8
	lastDay      = time.Now().Day()
)

func init() {
	alarmInfoMap = make(map[string]uint8)
}

func main() {

	if err := config.Load(); err != nil {
		panic(err)
	}

	cfg := config.Get()
	_ = cfg

	// 开启配置文件监控
	configFileWatcher := cfg.WatchConfigFileStatus()

	go func() {
		for {
			select {
			case <-configFileWatcher:
				log.Info("config changed")
			}
		}
	}()

	go cleanAlarmInfo()

	http.HandleFunc("/push", func(writer http.ResponseWriter, request *http.Request) {
		var buf bytes.Buffer
		defer request.Body.Close()
		io.Copy(&buf, request.Body)
		log.Debug(buf.String())
		checkLogMessage(*cfg, buf.String())
	})

	log.Info("listening on ", cfg.Address)
	if err := http.ListenAndServe(cfg.Address, nil); err != nil {
		log.Fatal(err)
	}
}

// 检查log信息是否匹配
func checkLogMessage(cfg config.Config, message string) {
	var logData logstash.LogData
	if err := json.Unmarshal([]byte(message), &logData); err != nil {
		log.Error(err)
		return
	}

	// 检查每个filter
	for _, v := range cfg.Filters {
		if v.Level != strings.ToUpper(logData.Level) {
			return
		}

		// 检查tag
		for _, t := range v.Tags {
			for _, lt := range logData.Tags {
				if t == lt {
					log.Info("match, will send email")
					// 发送邮件
					sendEmail(cfg, v, logData)
					break
				}
			}
		}
	}
}

func cleanAlarmInfo() {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case now := <-t.C:
			if now.Day() != lastDay {
				log.Info("clean old alarm message info")
				lastDay = now.Day()
				alarmInfoMap = make(map[string]uint8)
			}
		}
	}
}

func sendEmail(cfg config.Config, filter config.Filter, logData logstash.LogData) {
	now := time.Now()
	key := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprint(now.Day()) + logData.Host + "\n" + logData.Source + "\n" + logData.Message))
	count := alarmInfoMap[key]
	if count > cfg.MaxPerDay {
		log.Warn("max count send, key(base64):", key)
		return
	}

	subject := fmt.Sprintf("log error! time: %v\t source: %v", logData.Timestamp, logData.Source)
	email := mail.Email{MailInfo: cfg.Mail, Subject: subject, Data: logData, MailTemplate: "log.html", ToPersion: filter.ToPersion}
	if err := mail.SendEmail(email); err != nil {
		log.Error("send email error:", err, "\nto:", filter.ToPersion)
	} else {
		log.Info("send email success")
	}

}
