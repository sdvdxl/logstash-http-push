package mail

import (
	"bytes"
	"html/template"

	"github.com/log-dog/logstash-http-push/config"
	"github.com/log-dog/logstash-http-push/log"
	"gopkg.in/gomail.v2"
)

// Email 邮件信息
type Email struct {
	MailInfo     config.MailInfo
	ToPersion    []string
	Subject      string
	MailTemplate string
	Data         interface{}
}

// SendEmail 发送邮件
func SendEmail(email Email) error {
	log.Info("will send mail...")
	mailInfo := email.MailInfo
	m := gomail.NewMessage()
	m.SetHeader("From", mailInfo.Sender)
	m.SetHeader("To", email.ToPersion...)
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
