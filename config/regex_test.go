package config

import (
	"github.com/issue9/assert"
	"regexp"
	"testing"
)

func TestRegex(t *testing.T) {
	match := regexp.MustCompile("(mongo)|(rabbit)").MatchString(`违法无法 阿发
mongo啊啊阿飞
rabbit`)
	assert.True(t, match)
}
