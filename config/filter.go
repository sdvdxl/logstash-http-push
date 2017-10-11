package config

// Filter log 过滤
type Filter struct {
	Name           string   `json:"-"`
	IgnoreContains []string `json:"ignoreContains"` // 忽略的列表，普通字符串，如果包含其中一个则忽略，or 的关系
	lastMailIndex  int
	Levels         []string `json:"levels"`
	Tags           []string `json:"tags"`
	Ding           DingInfo `json:"ding"` // 钉钉 机器人token
	Mail           MailInfo `json:"mail"`
}

func (f *Filter) GetMail() MailSender {
	return f.Mail.Senders[f.lastMailIndex%len(f.Mail.Senders)]
}

func (f *Filter) GetNextMail() MailSender {
	f.lastMailIndex++
	return f.Mail.Senders[f.lastMailIndex%len(f.Mail.Senders)]
}
