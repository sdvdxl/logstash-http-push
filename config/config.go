package config

import (
	"sync"

	"github.com/sdvdxl/go-tools/errors"

	"log"

	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sdvdxl/go-tools/encrypt"
	"github.com/spf13/viper"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	configFile = "cfg.json"
)

// Cfg 配置文件
var (
	cfg    Config
	once   sync.Once
	inited = false
)

// Get 获取配置信息
func Get() *Config {
	if !inited {
		once.Do(load)
	}

	return &cfg
}

// Config 配置文件
type Config struct {
	Address   string             `json:"address"` //web 服务地址 ":5678"
	LogLevel  string             `json:"logLevel"`
	Filters   []*Filter          `json:"filters"`
	filterMap map[string]*Filter `json:"-"`
	TimeZone  int8               `json:"timeZone"` //时区，如果时间有偏移则加上时区，否则设置为0即可
}

const filterKeyPrefix = "filter-"

func (cfg *Config) GetFilter(tags []string, level string) []*Filter {
	matchFilters := make([]*Filter, 0, 10)
	if len(tags) == 0 && level == "" {
		if filter, exists := cfg.filterMap[filterKeyPrefix]; exists {
			return []*Filter{filter}
		}

		return []*Filter{}
	} else if len(tags) > 0 && level == "" {
		matchFilters = filterWithTags(cfg.Filters, tags)
	} else if len(tags) == 0 && level != "" {
		matchFilters = filterWithLevel(cfg.Filters, level)
	} else {
		matchFilters = filterWithTags(cfg.Filters, tags)
		matchFilters = filterWithLevel(matchFilters, level)
	}

	return matchFilters
}

func filterWithLevel(filters []*Filter, level string) []*Filter {
	result := make([]*Filter, 0, len(filters))
	for i := range filters {
		filter := filters[i]
		// 寻找匹配的 level
		for _, l := range filter.Levels {
			if strings.ToUpper(l) == strings.ToUpper(level) {
				result = append(result, filter)
				break
			}
		}
	}

	return result
}

func filterWithTags(filters []*Filter, tags []string) []*Filter {
	result := make([]*Filter, 0, len(filters))
	sort.Strings(tags)
	// 检查每个filter

	for i := range filters {
		tagMatchCount := 0
		filter := filters[i]
		// 寻找匹配的 Tags
		for _, filterTag := range filter.Tags {
			for _, logTag := range tags {
				if strings.ToUpper(logTag) == filterTag {
					tagMatchCount++
					break
				}
			}
		}

		if len(filter.Tags) == tagMatchCount {
			result = append(result, filter)
		}
	}

	return result
}

func (cfg Config) IsInited() bool {
	return inited
}

// Load 读取配置文件
func load() {
	viperReader := viper.New()
	viperReader.SetConfigFile(configFile)
	viperReader.SetConfigType("json")
	viperReader.OnConfigChange(func(event fsnotify.Event) {
		log.Println("file changed", event)
		if event.Op != fsnotify.Chmod {
			log.Println("config file changed, reloading...")
			readConfig(viperReader)
		}
	})
	viperReader.WatchConfig()
	readConfig(viperReader)
}

func readConfig(viperReader *viper.Viper) {
	log.Println("read config...")
	inited = false
	errors.Panic(viperReader.ReadInConfig())
	errors.Panic(viperReader.Unmarshal(&cfg))
	check()
}

func check() {
	log.Println("check config...")
	// 检查配置项目
	nameMap := make(map[string]bool)
	cfg.filterMap = make(map[string]*Filter)
	filters := cfg.Filters
	for i := range filters {
		filter := filters[i]

		//tag，
		{
			if filter.Tags == nil {
				filters[i].Tags = make([]string, 0)
			} else {
				for j := range filters[i].Tags {
					filters[i].Tags[j] = strings.ToUpper(strings.TrimSpace(filters[i].Tags[j]))
				}
			}
			sort.Strings(filter.Tags)
		}

		//处理日志级别，如果为 nil，变为空slice，并且将 level trim ，变大写
		{
			if filter.Levels == nil {
				filter.Levels = make([]string, 0)
				log.Println("filter ", filter.Name, " level not set, all levels will pass")
			} else {
				for j := range filter.Levels {
					filter.Levels[j] = strings.ToUpper(strings.TrimSpace(filters[i].Levels[j]))
					sort.Strings(filter.Levels)
				}
			}
		}

		{
			// filter name= filter-tags-levels，tags， levels 相同，则认为是同一个 filter
			filter.Name = filterKeyPrefix + fmt.Sprintf("%v-%v", strings.Join(filter.Tags, "-"), strings.Join(filter.Levels, "-"))
			if _, exists := nameMap[filter.Name]; exists {
				panic(fmt.Sprintf("filter  already exists, tags: %v, levels: %v", filter.Tags, filter.Levels))
			}

			nameMap[filter.Name] = true
		}

		// 钉钉
		{

			var tokens string
			for _, t := range filter.Ding.Senders {
				tokens += t.Token
			}
			filter.Ding.Name = fmt.Sprint(filter.Name, "-", encrypt.MD5([]byte(tokens)))

			if filter.Ding.MatchRegexText != "" {
				filter.Ding.MatchRegex = regexp.MustCompile(filter.Ding.MatchRegexText)
			}

			for j := range filter.Ding.Senders {
				d := filter.Ding.Senders[j]
				if d.Token == "" {
					panic(fmt.Sprint("filter ", filter.Name, "ding pos:", j, " token is empty, disabled"))
				}

			}
		}

		// mail
		{
			if len(filter.Mail.ToPersons) == 0 {
				panic(fmt.Sprint("filter ", filter.Name, "email toPersons is empty, disabled"))
			}

			for j := range filter.Mail.Senders {
				m := filter.Mail.Senders[j]

				if m.Sender == "" {
					panic(fmt.Sprint("filter ", filter.Name, "email pos", j, " sender is empty"))

				}

				if m.Password == "" {
					panic(fmt.Sprint("filter ", filter.Name, "email pos", j, " password is empty, disabled"))
				}

				if m.SMTP == "" {
					panic(fmt.Sprint("filter ", filter.Name, "email pos", j, " SMTP is empty, disabled"))
				}

			}

			if filter.Mail.Duration == 0 {
				log.Println("mail send duration default set to: 60s")
				filter.Mail.Duration = 60
			} else if filter.Mail.Duration < 5 {
				panic("duration is too small, must gte 5")
			}

			filter.Mail.Ticker = time.NewTicker(time.Second * time.Duration(filter.Mail.Duration))
		}
		log.Println("filter", filter.Name, "inited")
		cfg.filterMap[filter.Name] = filter

	}

	inited = true
	log.Println("config inited")
}
