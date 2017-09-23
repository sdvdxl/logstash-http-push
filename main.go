package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/sdvdxl/logstash-http-push/log"

	"strings"

	"fmt"

	"encoding/base64"

	"sync"

	"regexp"

	"github.com/sdvdxl/logstash-http-push/config"
	"github.com/sdvdxl/logstash-http-push/logstash"
	"github.com/sdvdxl/logstash-http-push/mail"
)

var (
	tagsMap      map[string]map[string]bool
	lastDay      = time.Now().Day()
	alarmInfo    = &AlarmInfo{}
	messageRegex *regexp.Regexp
)

// AlarmInfo 告警记录
type AlarmInfo struct {
	lock         sync.Mutex
	alarmInfoMap map[string]uint64
}

// GetValues 获取数据
func (a *AlarmInfo) GetValues() map[string]uint64 {
	defer a.lock.Unlock()
	a.lock.Lock()
	r := make(map[string]uint64)
	for k, v := range a.alarmInfoMap {
		r[k] = v
	}

	return r
}

// GetAndAnd 获取并且加1
func (a *AlarmInfo) GetAndAnd(key string) uint64 {
	defer a.lock.Unlock()
	a.lock.Lock()
	c := a.alarmInfoMap[key]
	a.alarmInfoMap[key] = c + 1
	return c
}

// Reset 重置数据
func (a *AlarmInfo) Reset() {
	defer a.lock.Unlock()
	a.lock.Lock()

	a.alarmInfoMap = make(map[string]uint64)
}

func init() {
	alarmInfo.alarmInfoMap = make(map[string]uint64)
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
				messageRegex = regexp.MustCompile(cfg.Match)
				log.Info("config changed")
			}
		}
	}()

	go cleanAlarmInfo()

	http.HandleFunc("/push", func(writer http.ResponseWriter, request *http.Request) {
		var buf bytes.Buffer
		defer request.Body.Close()
		io.Copy(&buf, request.Body)
		// log.Debug(buf.String())
		matchCount := checkLogMessage(*cfg, buf.String())
		writer.Write([]byte(fmt.Sprint(matchCount)))
	})

	log.Info("listening on ", cfg.Address)
	if err := http.ListenAndServe(cfg.Address, nil); err != nil {
		log.Fatal(err)
	}
}

// 检查log信息是否匹配
func checkLogMessage(cfg config.Config, message string) uint64 {
	var logData logstash.LogData
	if err := json.Unmarshal([]byte(message), &logData); err != nil {
		log.Error(err)
		return 0
	}

	logData.Timestamp = logData.Timestamp.Add(time.Hour * time.Duration(cfg.TimeZone))

	// 检查每个filter
	for _, v := range cfg.Filters {
		if v.Level != strings.ToUpper(logData.Level) {
			return 0
		}

		// 检查tag
		count := 0
		size := len(v.Tags)
		for _, t := range v.Tags {
			for _, lt := range logData.Tags {
				if t == lt {
					count++
					if count == size {
						// 发送邮件
						now := time.Now()
						key := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprint(now.Day()) + logData.Host + "\n" + logData.Source + "\n" + getMatchMessage(cfg, logData.Message)))

						count := alarmInfo.GetAndAnd(key)
						if count >= cfg.MaxPerDay {
							return count + 1
						}

						if cfg.IsSendEmail {
							go sendEmail(cfg, v, logData)
						}
						return count + 1
					}
				}
			}
		}
	}
	return 0
}

func cleanAlarmInfo() {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case now := <-t.C:
			if now.Day() != lastDay {
				log.Info("clean old alarm message info")
				lastDay = now.Day()

				log.Info("old alarm Info:")
				for k, v := range alarmInfo.GetValues() {
					bs, _ := base64.StdEncoding.DecodeString(k)
					log.Info(string(bs), " count: ", v)
				}
				alarmInfo.Reset()
			}
		}
	}
}

func sendEmail(cfg config.Config, filter config.Filter, logData logstash.LogData) {
	subject := fmt.Sprintf("log error! time: %v\t source: %v", logData.Timestamp, logData.Source)
	email := mail.Email{MailInfo: cfg.Mail, Subject: subject, Data: logData, MailTemplate: "log.html", ToPersion: filter.ToPersion}
	if err := mail.SendEmail(email); err != nil {
		log.Error("send email error:", err, "\nto:", filter.ToPersion)
	} else {
		log.Info("send email success")
	}

}

func getMatchMessage(cfg config.Config, message string) string {
	str := messageRegex.FindString(message)
	if cfg.Negate {
		return strings.Replace(message, str, "", 1)
	}

	return str
}

var s = `\d{4}(\-\d{2}){2}T(\d{2}:){2}\d{2}\.\d{3}\+\d+\s+\w+\s+\[[\w\-]+\]\s+`
