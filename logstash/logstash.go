package logstash

import "time"

//{"offset":656674,"level":"ERROR","input_type":"log","source":"/data/logs/console.2017-02-10.log","message":"2017-02-10T16:21:28.942+0800 ERROR [http-nio-8080-exec-5] org.apache.velocity.log:96 - ResourceManager : unable to find resource '404.json.vm' in any resource loader. ","type":"log","tags":["smartmatrix","console","beats_input_codec_plain_applied"],"@timestamp":"2017-02-10T08:21:28.942Z","@version":"1","beat":{"hostname":"ubuntu","name":"ubuntu","version":"5.1.1"},"host":"ubuntu","input_timestamp":"2017-02-22T04:06:07.197Z"}

// LogData logstash output
type LogData struct {
	Level     string    `json:"level"`
	InputType string    `json:"input_type"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"@timestamp"`
	Beat      Beat      `json:"beat"`
	Tags      []string  `json:"tags"`
}

type Beat struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Hostname string `json:"hostname"`
}
