package config

import "regexp"

type DingInfo struct {
	Name           string           `json:"-"`
	Token          string           `json:"token"`
	Enable         bool             `json:"enable"`
	MatchRegexText []string         `json:"matchRegex"`
	MatchRegex     []*regexp.Regexp `json:"-"`
}
