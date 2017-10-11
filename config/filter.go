package config

// Filter log 过滤
type Filter struct {
	Name           string   `json:"-"`
	IgnoreContains []string `json:"ignoreContains"` // 忽略的列表，普通字符串，如果包含其中一个则忽略，or 的关系
	lastMailIndex  int
	Levels         []string   `json:"levels"`
	Tags           []string   `json:"tags"`
	Dings          []DingInfo `json:"dings"` // 钉钉 机器人token
	Mails          []MailInfo `json:"mails"`
}

func (f *Filter) GetMail() MailInfo {
	return f.Mails[f.lastMailIndex%len(f.Mails)]
}

func (f *Filter) GetNextMail() MailInfo {
	f.lastMailIndex++

	return f.Mails[f.lastMailIndex%len(f.Mails)]
}
