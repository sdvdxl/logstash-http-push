package mail

import (
	"bytes"
	"html/template"

	"github.com/sdvdxl/logstash-http-push/log"
	"gopkg.in/gomail.v2"
)

// Email 邮件信息
type Email struct {
	MailInfo     MailInfo
	ToPerson     []string
	Subject      string
	MailTemplate string
	Data         interface{}
}

// MailInfo 邮件信息
type MailInfo struct {
	SMTP     string `json:"smtp"`
	Port     int    `json:"port"`
	Sender   string `json:"sender"`
	Password string `json:"password"`
}

// SendEmail 发送邮件
func SendEmail(email Email) error {
	log.Info("sendding mail...")
	mailInfo := email.MailInfo
	m := gomail.NewMessage()
	m.SetHeader("From", mailInfo.Sender)
	m.SetHeader("To", email.ToPerson...)
	m.SetHeader("Subject", email.Subject)

	d := gomail.NewDialer(mailInfo.SMTP, mailInfo.Port, mailInfo.Sender, mailInfo.Password)
	tmpl, err := template.ParseFiles("templates/" + email.MailTemplate)
	if err != nil {
		return err
	}

	var contents bytes.Buffer
	if err = tmpl.Execute(&contents, email.Data); err != nil {
		return err
	}

	m.SetBody("text/html", contents.String())

	return d.DialAndSend(m)
}
