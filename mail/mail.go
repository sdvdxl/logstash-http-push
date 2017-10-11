package mail

import (
	"github.com/sdvdxl/logstash-http-push/config"
	"github.com/sdvdxl/logstash-http-push/log"
	"gopkg.in/gomail.v2"
)

// Email 邮件信息
type Email struct {
	MailSender config.MailSender
	ToPerson   []string
	Subject    string
	Message    string
	Data       interface{}
}

// SendEmail 发送邮件
func SendEmail(email Email) error {
	log.Info("sending mail to:", email.ToPerson)
	mailInfo := email.MailSender
	m := gomail.NewMessage()
	m.SetHeader("From", mailInfo.Sender)
	m.SetHeader("To", email.ToPerson...)
	m.SetHeader("Subject", email.Subject)

	d := gomail.NewDialer(mailInfo.SMTP, mailInfo.Port, mailInfo.Sender, mailInfo.Password)

	m.SetBody("text/html", email.Message)

	return d.DialAndSend(m)
}
