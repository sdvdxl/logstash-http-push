package config

// MailInfo 邮件信息
type MailInfo struct {
	Name      string   `json:"-"`
	Enable    bool     `json:"enable"`
	SMTP      string   `json:"smtp"`
	Port      int      `json:"port"`
	Sender    string   `json:"sender"`
	Password  string   `json:"password"`
	ToPersons []string `json:"toPersons"`
}
