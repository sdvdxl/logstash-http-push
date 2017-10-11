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
	"html/template"
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
	log.Init(cfg)

	for _, filter := range cfg.Filters {

		// 配置 ticker
		if filter.Mail.Ticker != nil {
			go func() {
				log.Info("init filter ", filter.Name, "ticker , duration", filter.Mail.Duration)

				for {
					select {
					case <-filter.Mail.Ticker.C:
						func() {
							defer filter.Mail.Lock.Unlock()
							log.Debug("ticker report")
							filter.Mail.Lock.Lock()
							if len(filter.Mail.MailMessages) == 0 {
								return
							}

							sendSuccess := false
							ding := getDing(filter)
							var message, errMsgs string
							for range filter.Mail.Senders { // 如果失败，循环发送，直到配置的所有邮箱有成功的，或者全部失败
								title := fmt.Sprint(filter.Tags, filter.Mail.Duration, "秒内邮件聚合")

								mailSender := filter.GetMail()
								message = strings.Join(filter.Mail.MailMessages, "<br><br><hr>")
								filter.Mail.MailMessages = make([]string, 0, 10)

								email := mail.Email{MailSender: mailSender, Subject: title, Message: message, ToPerson: filter.Mail.ToPersons}
								if err := mail.SendEmail(email); err != nil {
									errMsg := fmt.Sprint("send email error:", err, "\nsender:", filter.GetMail().Sender, "\nto:", filter.Mail.ToPersons)
									errMsgs += errMsg + "\n"
									log.Error(errMsg)
									mailSender = filter.GetNextMail()
								} else {
									sendSuccess = true
									log.Info("send email success")
									break
								}
							}

							if !sendSuccess && ding != nil {
								ding := getDing(filter)
								senders := ""
								for _, m := range filter.Mail.Senders {
									senders += m.Sender + " "
								}
								ding.Push(fmt.Sprintf("所有 mail 都发送失败，，失败信息: \n\n%v,请检查发送频率或者邮件信息，下面是发送失败的错误：\n\n %v", errMsgs, message))
							}
						}()
					}
				}
			}()
		}

		// 配置钉钉
		if filter.Ding.Enable {

			for _, d := range filter.Ding.Senders {
				ding := &dinghook.DingQueue{Interval: 3, Limit: 1, Title: "【告警】", AccessToken: d.Token}
				ding.Init()
				go ding.Start()
				dingMap[d.Token] = ding
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
		if !filter.Ding.Enable {
			log.Debug("ding ", filter.Ding.Name, " is disabled")
			continue
		}
		msg := logData.Message
		for r := range filter.Ding.MatchRegex {
			if filter.Ding.MatchRegex[r].MatchString(msg) {
				idx := strings.Index(msg, " at")

				if idx > 0 {
					msg = msg[:idx]
				}
				title := logData.Source[strings.Index(logData.Source, logPathPrefix)+logPathPrefixLen : strings.Index(logData.Source, ".")]

				for _, d := range filter.Ding.Senders {
					ding := dingMap[d.Token]
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

		if !filter.Mail.Enable {
			continue
		}

		// 如果 ticker 不是 nil，则定时发送
		message := getMessage(logData)
		func() {
			defer filter.Mail.Lock.Unlock()
			filter.Mail.Lock.Lock()
			filter.Mail.MailMessages = append(filter.Mail.MailMessages, message)
		}()

	}
}

func getMessage(logdata logstash.LogData) string {
	tmpl, err := template.ParseFiles("templates/log.html")
	errors.Panic(err)

	var contents bytes.Buffer
	errors.Panic(tmpl.Execute(&contents, logdata))
	return contents.String()
}
