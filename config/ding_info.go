package config

import "regexp"

type DingInfo struct {
	// 跟现在比较，如果超过了这个时间则忽略不发送
	IgnoreIfGtSecs int64          `json:"ignoreIfGtSecs" mapstructure:"ignoreIfGtSecs"`
	Name           string         `json:"-" mapstructure:"-"`
	Enable         bool           `json:"enable" mapstructure:"enable"`
	MatchRegexText string         `json:"matchRegex" mapstructure:"matchRegex"`
	MatchRegex     *regexp.Regexp `json:"-" mapstructure:"-"`
	Senders        []DingSender   `json:"-" mapstructure:"senders"`
}

type DingSender struct {
	Token string `json:"token" mapstructure:"token"`
}
