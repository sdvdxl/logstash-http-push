package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"strings"
	"sync"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sdvdxl/dinghook"
	"github.com/sdvdxl/logstash-http-push/config"
	"github.com/sdvdxl/logstash-http-push/log"
	"github.com/sdvdxl/logstash-http-push/logstash"
	"github.com/sdvdxl/logstash-http-push/mail"
)

const (
	// logPathPrefix 日志路径前缀
	logPathPrefix    = "/data/logs/"
	logPathPrefixLen = len(logPathPrefix)
	splitChar        = "@"
)

var (
	lastDay   = time.Now().Day()
	alarmInfo = &AlarmInfo{}
	dingMap   map[string](*dinghook.DingQueue)
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
	engine := echo.New()
	engine.Use(middleware.Logger())
	engine.Use(middleware.Recover())

	cfg := config.Get()

	// 配置钉钉

	for _, f := range cfg.Filters {
		if len(f.Ding) > 0 {
			ding := &dinghook.DingQueue{Interval: 3, Limit: 1, Title: "【告警】", AccessToken: f.Ding}
			ding.Init()
			go ding.Start()
			dingMap[strings.Join(f.Tags, splitChar)] = ding
		}
	}

	go cleanAlarmInfo()

	engine.POST("/push", func(c echo.Context) error {
		var buf bytes.Buffer
		defer c.Request().Body.Close()
		io.Copy(&buf, c.Request().Body)
		log.Debug(buf.String())

		matchCount := checkLogMessage(cfg, buf.String())
		return c.String(http.StatusOK, fmt.Sprint(matchCount))
	})
	engine.Server.Addr = cfg.Address
	gracehttp.Serve(engine.Server)
}

// 检查log信息是否匹配
func checkLogMessage(cfg *config.Config, message string) uint64 {
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

		dingTags := make([]string, 0, 10)
		// 检查tag
		count := 0
		size := len(v.Tags)
		for _, t := range v.Tags {
			for _, lt := range logData.Tags {
				if t == lt {
					count++
					dingTags = append(dingTags, t)
					if count == size {

						// 发送邮件
						now := time.Now()
						key := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprint(now.Day()) + logData.Host + "\n" + logData.Source + "\n" + getMatchMessage(cfg, logData.Message)))

						count := alarmInfo.GetAndAdd(key)
						if count >= cfg.MaxPerDay {
							log.Warn("max per day reached，not send email， max count ：", cfg.MaxPerDay, " current count：", count)
							return count
						}

						for _, ig := range v.Ignores {
							if strings.Contains(message, ig) {
								return count
							}
						}

						// 发送钉钉
						go sendDing(v, logData)

						if cfg.IsSendEmail {
							go sendEmail(v, logData)
						}
						return count
					}
				}
			}
		}
	}
	return 0
}

func getDing(filter *config.Filter) *dinghook.DingQueue {
	return dingMap[strings.Join(filter.Tags, splitChar)]
}

func sendDing(filter *config.Filter, logData logstash.LogData) {
	msg := logData.Message
	if strings.Contains(msg, "mongodb") || strings.Contains(msg, "AmqpConnectException") {
		idx := strings.Index(msg, " at")

		if idx > 0 {
			msg = msg[:idx]
		}
		title := logData.Source[strings.Index(logData.Source, logPathPrefix)+logPathPrefixLen : strings.Index(logData.Source, ".")]
		ding := getDing(filter)
		if ding != nil {
			ding.PushMessage(dinghook.SimpleMessage{Title: title, Content: title + " \n\n " + msg})
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

func sendEmail(filter *config.Filter, logData logstash.LogData) {

	title := logData.Source[strings.Index(logData.Source, logPathPrefix)+logPathPrefixLen : strings.Index(logData.Source, ".")]

	subject := fmt.Sprintf("❌ %v\t time: %v", title, logData.Timestamp)
	mailInfo := filter.GetMail()
	sendSuccess := false
	ding := getDing(filter)
	for range filter.Mails { // 如果失败，循环发送，直到配置的所有邮箱有成功的，或者全部失败
		email := mail.Email{MailInfo: mailInfo, Subject: subject, Data: logData, MailTemplate: "log.html", ToPerson: filter.ToPerson}
		if err := mail.SendEmail(email); err != nil {
			errMsg := fmt.Sprint("send email error:", err, "\nto:", filter.ToPerson)
			log.Error(errMsg)
			mailInfo = filter.GetNextMail()
		} else {
			sendSuccess = true
			log.Info("send email success")
			break
		}
	}

	if !sendSuccess && ding != nil {
		ding := getDing(filter)
		ding.Push("所有 mail 都发送失败，请检查发送频率或者邮件信息，下面是发送失败的错误：\n" + logData.Message)
	}

}

func getMatchMessage(cfg *config.Config, message string) string {
	str := cfg.MatchRegex.FindString(message)
	if cfg.Negate {
		return strings.Replace(message, str, "", 1)
	}

	return str
}

var s = `\d{4}(\-\d{2}){2}T(\d{2}:){2}\d{2}\.\d{3}\+\d+\s+\w+\s+\[[\w\-]+\]\s+`
