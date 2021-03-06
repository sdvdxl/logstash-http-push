package logstash

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

const (
	text = `{"offset":656674,"level":"ERROR","input_type":"log","source":"/data/logs/console.2017-02-10.log","message":"2017-02-10T16:21:28.942+0800 ERROR [http-nio-8080-exec-5] org.apache.velocity.log:96 - ResourceManager : unable to find resource '404.json.vm' in any resource loader. ","type":"log","tags":["smartmatrix","console","beats_input_codec_plain_applied"],"@timestamp":"2017-02-10T08:21:28.942Z","@version":"1","beat":{"hostname":"ubuntu","name":"ubuntu","version":"5.1.1"},"host":"ubuntu","input_timestamp":"2017-02-22T04:06:07.197Z"}`

	message   = "2017-02-10T16:21:28.942+0800 ERROR [http-nio-8080-exec-5] org.apache.velocity.log:96 - ResourceManager : unable to find resource '404.json.vm' in any resource loader. "
	matchStr  = `2017-02-10T16:21:28.942+0800 ERROR [http-nio-8080-exec-5] `
	resultStr = `org.apache.velocity.log:96 - ResourceManager : unable to find resource '404.json.vm' in any resource loader. `
)

func TestParseJson(t *testing.T) {
	var log LogData
	if err := json.Unmarshal([]byte(text), &log); err != nil {
		t.Fail()
	}

	fmt.Println(log)
}

func TestMatchRegex(t *testing.T) {
	r := regexp.MustCompile(`\d{4}(\-\d{2}){2}T(\d{2}:){2}\d{2}\.\d{3}\+\d+\s+\w+\s+\[[\w\-]+\]\s+`)
	str := r.FindString(message)
	if str != matchStr {
		fmt.Println("not match")
		t.Fail()
	}

	if strings.Replace(message, str, "", 1) != resultStr {
		t.Fail()
	}
}
