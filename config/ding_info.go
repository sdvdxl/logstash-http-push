package config

import "regexp"

type DingInfo struct {
	Name           string         `json:"-" mapstructure:"-"`
	Enable         bool           `json:"enable" mapstructure:"enable"`
	MatchRegexText string         `json:"matchRegex" mapstructure:"matchRegex"`
	MatchRegex     *regexp.Regexp `json:"-" mapstructure:"-"`
	Senders        []DingSender   `json:"-" mapstructure:"senders"`
}

type DingSender struct {
	Token string `json:"token" mapstructure:"token"`
}
