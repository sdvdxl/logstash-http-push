package config

import "regexp"

type DingInfo struct {
	Name           string           `json:"-"`
	Enable         bool             `json:"enable"`
	MatchRegexText []string         `json:"matchRegex"`
	MatchRegex     []*regexp.Regexp `json:"-"`
	Senders        []DingSender     `json:"-"`
}

type DingSender struct {
	Token string `json:"token"`
}
