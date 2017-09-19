package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/log-dog/logstash-http-push/log"
	"github.com/tylerb/graceful"

	"github.com/sdvdxl/dinghook"

	"strings"

	"fmt"

	"encoding/base64"

	"sync"

	"regexp"

	"github.com/log-dog/logstash-http-push/config"
	"github.com/log-dog/logstash-http-push/logstash"
	"github.com/log-dog/logstash-http-push/mail"
)

const (
	LOG_PREFIX     = "/data/logs/"
	LOG_PREFIX_LEN = len(LOG_PREFIX)
)

var (
	tagsMap      map[string]map[string]bool
	lastDay      = time.Now().Day()
	alarmInfo    = &AlarmInfo{}
	messageRegex *regexp.Regexp
	dingMap      map[string](*dinghook.DingQueue)
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

// GetAndAdd 获取并且加1
func (a *AlarmInfo) GetAndAdd(key string) uint64 {
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
	dingMap = make(map[string](*dinghook.DingQueue))
}

func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}

	cfg := config.Get()

	// 配置钉钉

	for _, f := range cfg.Filters {
		if len(f.Ding) > 0 {
			ding := &dinghook.DingQueue{Interval: 3, Limit: 1, Title: "【告警】", AccessToken: f.Ding}
			ding.Init()
			go ding.Start()
			sort.Strings(f.Tags)
			dingMap[strings.Join(f.Tags, "-")] = ding
		}
	}

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

	engine := echo.New()
	engine.Use(middleware.Recover())
	// engine.Use(middleware.Logger())

	engine.Server.Addr = cfg.Address
	server := &graceful.Server{Timeout: time.Second * 10, Server: engine.Server, Logger: graceful.DefaultLogger()}

	go cleanAlarmInfo()

	engine.POST("/push", func(c echo.Context) error {
		var buf bytes.Buffer
		defer c.Request().Body.Close()
		io.Copy(&buf, c.Request().Body)
		// log.Debug(buf.String())
		matchCount := checkLogMessage(*cfg, buf.String())
		return c.JSON(http.StatusOK, fmt.Sprint(matchCount))
	})

	log.Info("listening on ", cfg.Address)
	if err := server.ListenAndServe(); err != nil {
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

	// 如果是mongo相关，发送到钉钉
	go func() {
		msg := logData.Message
		if strings.Contains(msg, "mongodb") || strings.Contains(msg, "AmqpConnectException") {
			idx := strings.Index(msg, " at")

			if idx > 0 {
				msg = msg[:idx]
			}
			title := logData.Source[strings.Index(logData.Source, LOG_PREFIX)+LOG_PREFIX_LEN : strings.Index(logData.Source, ".")]
			sort.Strings(logData.Tags)
			ding := dingMap[strings.Join(logData.Tags, "-")]
			ding.PushMessage(dinghook.SimpleMessage{Title: title, Content: title + " \n\n " + msg})
		}
	}()

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

						count := alarmInfo.GetAndAdd(key)
						if count >= cfg.MaxPerDay {
							return count
						}

						for _, ig := range v.Ignores {
							if strings.Contains(message, ig) {
								return count
							}
						}

						if cfg.IsSendEmail {
							go sendEmail(cfg, v, logData)
						}
						return count
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
	title := logData.Source[strings.Index(logData.Source, LOG_PREFIX)+LOG_PREFIX_LEN : strings.Index(logData.Source, ".")]

	subject := fmt.Sprintf("❌ %v\t time: %v", title, logData.Timestamp)
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
