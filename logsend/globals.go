package logsend

import (
	logpkg "log"
	"os"
	"runtime/pprof"
	"strconv"
)

// 注册新的发送者
func RegisterNewSender(name string, init func(interface{}), get func() Sender) {
	sender := &SenderRegister{
		init: init,
		get:  get,
	}
	Conf.registeredSenders[name] = sender
	Conf.Logger.Println("register sender:", name)
	return
}

// 发送者类型
type SenderRegister struct {
	init        func(interface{})
	get         func() Sender
	initialized bool
}

func (self *SenderRegister) Init(val interface{}) {
	self.init(val)
	self.initialized = true
}

// 配置类型
type Configuration struct {
	WatchDir          string
	ContinueWatch     bool
	Debug             bool
	Memprofile        string
	Logger            *logpkg.Logger
	DryRun            bool
	ReadWholeLog      bool
	ReadOnce          bool
	memprofile        *os.File
	Cpuprofile        string
	cpuprofile        *os.File
	registeredSenders map[string]*SenderRegister
}

// 创建全局配置变量
var Conf = &Configuration{
	WatchDir:          "",
	Memprofile:        "",
	Cpuprofile:        "",
	Logger:            logpkg.New(os.Stderr, "", logpkg.Ldate|logpkg.Ltime|logpkg.Lshortfile),
	registeredSenders: make(map[string]*SenderRegister),
}

var (
	// 原始配置变量,使用pipe处理日志时用到
	rawConfig = make(map[string]interface{}, 0)
)

// 内存分析
func mempprof() {
	if Conf.memprofile == nil {
		Conf.memprofile, _ = os.Create(Conf.Memprofile)
	}
	pprof.WriteHeapProfile(Conf.memprofile)
}

func debug(msg ...interface{}) {
	if !Conf.Debug {
		return
	}
	Conf.Logger.Printf("debug: %+v", msg)
}

// 类型转换
func i2float64(i interface{}) float64 {
	switch i.(type) {
	case string:
		val, _ := strconv.ParseFloat(i.(string), 32)
		return val
	case int:
		return float64(i.(int))
	case float64:
		return i.(float64)
	}
	panic(i)
}

func i2int(i interface{}) int {
	switch i.(type) {
	case string:
		val, _ := strconv.ParseFloat(i.(string), 32)
		return int(val)
	case int:
		return i.(int)
	case float64:
		return int(i.(float64))
	}
	panic(i)
}
