package main

import (
	"bytes"
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
	"github.com/sdvdxl/go-tools/errors"
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

	for _, filter := range cfg.Filters {
		// 配置钉钉
		for _, d := range filter.Dings {
			if d.Enable {
				ding := &dinghook.DingQueue{Interval: 3, Limit: 1, Title: "【告警】", AccessToken: d.Token}
				ding.Init()
				go ding.Start()
				dingMap[d.Name] = ding
			}
		}

	}

	engine.POST("/push", func(c echo.Context) error {
		var buf bytes.Buffer
		defer c.Request().Body.Close()
		io.Copy(&buf, c.Request().Body)
		log.Debug(buf.String())
		logData, err := convertMessageToData(cfg, buf.String())
		errors.Panic(err)
		send(cfg, logData)
		return c.String(http.StatusOK, "")
	})
	engine.Server.Addr = cfg.Address
	log.Info("listen on ", cfg.Address)
	gracehttp.Serve(engine.Server)
}

// 将 message 转换成对象
func convertMessageToData(cfg *config.Config, message string) (*logstash.LogData, error) {
	var logData logstash.LogData
	if err := json.Unmarshal([]byte(message), &logData); err != nil {
		log.Error(err)
		return nil, err
	}

	logData.Timestamp = logData.Timestamp.Add(time.Hour * time.Duration(cfg.TimeZone))
	return &logData, nil
}

// 检查log信息是否匹配
func send(cfg *config.Config, logData *logstash.LogData) {

	matchFilter := cfg.GetFilter(logData.Tags, logData.Level)
	go sendDing(matchFilter, *logData)
	go sendEmail(matchFilter, *logData)
}

func getDing(filter *config.Filter) *dinghook.DingQueue {
	return dingMap[strings.Join(filter.Tags, splitChar)]
}

func sendDing(filters []*config.Filter, logData logstash.LogData) {
	for _, filter := range filters {
		for _, d := range filter.Dings {
			if !d.Enable {
				log.Debug("ding ", d.Name, " is disabled")
				continue
			}

			msg := logData.Message
			for _, r := range d.MatchRegex {
				if r.MatchString(msg) {
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

		}
	}

}

func sendEmail(filters []*config.Filter, logData logstash.LogData) {
	for _, filter := range filters {

		title := logData.Source[strings.Index(logData.Source, logPathPrefix)+logPathPrefixLen : strings.Index(logData.Source, ".")]

		subject := fmt.Sprintf("❌ %v\t time: %v", title, logData.Timestamp)
		mailInfo := filter.GetMail()
		sendSuccess := false
		ding := getDing(filter)
		var errMsgs string
		for _, fm := range filter.Mails { // 如果失败，循环发送，直到配置的所有邮箱有成功的，或者全部失败
			if !fm.Enable {
				continue
			}

			email := mail.Email{MailInfo: mailInfo, Subject: subject, Data: logData, MailTemplate: "log.html", ToPerson: fm.ToPersons}
			if err := mail.SendEmail(email); err != nil {
				errMsg := fmt.Sprint("send email error:", err, "\nsender:", filter.GetMail().Sender, "\nto:", fm.ToPersons)
				errMsgs += errMsg + "\n"
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
			senders := ""
			for _, m := range filter.Mails {
				senders += m.Sender + " "
			}
			ding.Push(fmt.Sprintf("所有 mail 都发送失败，，失败信息: \n%v,请检查发送频率或者邮件信息，下面是发送失败的错误：\n %v", errMsgs, logData.Message))
		}

	}
}
