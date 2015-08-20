package logsend

import (
	"bufio"
	"flag"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Using PIPE 模式
// ProcessStdin read data from input for process it
// ProcessStdin 从输入数据读取处理
func ProcessStdin() error {
	var rules []*Rule
	// 方式判断
	if rawConfig["config"].(flag.Value).String() != "" { 
		// with config.json:
		// eg: logsend -config=config.json /logs
		groups, err := LoadConfigFromFile(rawConfig["config"].(flag.Value).String())
		if err != nil {
			Conf.Logger.Fatalf("can't load config %+v", err)
		}
		for _, group := range groups {
			for _, rule := range group.Rules {
				rules = append(rules, rule)
			}
		}
	} else {
		// without config.json(这种方式只有一个sender有效)
		// eg: tail -F /logs/*.log |logsend -influx-dbname test -influx-host 'hosta:4444' -regex='\d+'
		// TODO: move this to separate method
		if rawConfig["regex"].(flag.Value).String() == "" {
			Conf.Logger.Fatalln("regex not set")
		}
		// 匹配发送者
		matchSender := regexp.MustCompile(`(\w+)-host`)
		var sender Sender
		for key, val := range rawConfig {
			match := matchSender.FindStringSubmatch(key)
			if len(match) == 0 || val.(flag.Value).String() == "" {
				continue
			}
			// 匹配sender并初始化设置相关参数
			if register, ok := Conf.registeredSenders[match[1]]; ok {
				conf := make(map[string]interface{})
				for key, val := range rawConfig {
					newKey := key
					if ok, _ := regexp.MatchString(match[1], key); ok {
						newKey = strings.Split(key, match[1]+"-")[1]
					}
					switch val.(flag.Value).String() {
					default:
						conf[newKey] = interface{}(val.(flag.Value).String())
					case "true", "false":
						b, err := strconv.ParseBool(val.(flag.Value).String())
						if err != nil {
							Conf.Logger.Fatalln(err)
						}
						conf[newKey] = interface{}(b)
					}
				}
				register.Init(conf)
				sender = register.get()
				sender.SetConfig(conf)
				break
			}
		}
		// 创建规则
		rule, err := NewRule(rawConfig["regex"].(flag.Value).String())
		if err != nil {
			panic(err)
		}
		rule.senders = []Sender{sender}
		rules = append(rules, rule)
	}

	// 读取标准输入流
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			break
		}
		checkLineRules(&line, rules)
	}
	return nil
}
