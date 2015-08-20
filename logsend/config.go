package logsend

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
)

// 加载原始置(已定义的flag参数)
func LoadRawConfig(f *flag.Flag) {
	rawConfig[f.Name] = f.Value
}

// 从文件加载配置
func LoadConfigFromFile(fileName string) (groups []*Group, err error) {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// 读取配置文件到rawConfig对象
	rawConfig, err := ioutil.ReadAll(file)
	if err != nil {
		Conf.Logger.Fatalln(err)
	}
	return LoadConfig(rawConfig)
}

// 加载配置
func LoadConfig(rawConfig []byte) (groups []*Group, err error) {
	config := make(map[string]interface{})
	// 反序列化JSON对象
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}

	// 循环遍历以实现的发送者与配置匹配,找到后初始化(可以理解为初始化数据源)
	for sender, register := range Conf.registeredSenders {
		if val, ok := config[sender]; ok {
			register.Init(val)
		}
	}

	// "组"配置初始化
	for _, groupConfig := range config["groups"].([]interface{}) {
		group := &Group{}
		// mask(文件名)
		if group.Mask, err = regexp.Compile(groupConfig.(map[string]interface{})["mask"].(string)); err != nil {
			Conf.Logger.Fatalln(err)
		}
		// 规则(日志内容)
		for _, groupRule := range groupConfig.(map[string]interface{})["rules"].([]interface{}) {
			// regex, err := regexp.Compile()
			if err != nil {
				Conf.Logger.Fatalln(err)
			}
			senders := make([]Sender, 0)
			for senderName, register := range Conf.registeredSenders {
				// not load rules to not initilized senders
				// 对没有初始化的发送者不加载规则
				if register.initialized != true {
					continue
				}
				// 设置针对此规则的配置
				if val, ok := groupRule.(map[string]interface{})[senderName].(interface{}); ok {
					sender := register.get()
					if err = sender.SetConfig(val); err != nil {
						Conf.Logger.Fatalln(err)
					}
					senders = append(senders, sender)
				}
			}
			// 创建此规则并关联发送者
			rule, err := NewRule(groupRule.(map[string]interface{})["regexp"].(string))
			if err != nil {
				panic(err)
			}
			rule.senders = senders
			// 添加到当前组的规则里
			group.Rules = append(group.Rules, rule)
		}
		// 添加到总的组
		groups = append(groups, group)
	}
	return
}
