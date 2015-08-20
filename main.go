package main

import (
	"flag"
	"fmt"
	"github.com/rosetears/logsend/logsend"
	logpkg "log"
	"os"
)

const (
	// VERSION definition
	VERSION = "1.7.1"
)

var (
	// watch的目录，已过期，现在使用最有一个参数作为目录
	watchDir = flag.String("watch-dir", "", "deprecated, simply add the directory as an argument, in the end")
	// config.json文件的路径
	config = flag.String("config", "", "path to config.json file")
	// 检查配置文件
	check = flag.Bool("check", false, "check config.json")
	// 打开debug信息
	debug = flag.Bool("debug", false, "turn on debug messages")
	// 继续watch目录新文件
	continueWatch = flag.Bool("continue-watch", false, "watching folder for new files")
	// 程序日志
	logFile = flag.String("log", "", "log file")
	// 试运行，不发送数据
	dryRun = flag.Bool("dry-run", false, "not send data")
	// 内存分析输出文件
	memprofile = flag.String("memprofile", "", "memory profiler")
	// 最大使用cpu个数
	maxprocs = flag.Int("maxprocs", 0, "max count of cpu")
	// 读取整个日志
	readWholeLog = flag.Bool("read-whole-log", false, "read whole logs")
	// 读取日志一次并退出
	readOnce = flag.Bool("read-once", false, "read logs once and exit")
	// 正则表达式
	regex = flag.String("regex", "", "regex rule")
	// 显示版本信息
	version = flag.Bool("version", false, "show version number")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 打印版本信息
	if *version {
		fmt.Printf("logsend version %v\n", VERSION)
		os.Exit(0)
	}

	// 日志文件
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Errorf("Failed to open log file: %+v\n", err)
		}
		defer file.Close()
		logsend.Conf.Logger = logpkg.New(file, "", logpkg.Ldate|logpkg.Ltime|logpkg.Lshortfile)
	}

	// 相关参数绑定到全局配置
	logsend.Conf.Debug = *debug
	logsend.Conf.ContinueWatch = *continueWatch
	logsend.Conf.WatchDir = *watchDir
	logsend.Conf.Memprofile = *memprofile
	logsend.Conf.DryRun = *dryRun
	logsend.Conf.ReadWholeLog = *readWholeLog
	logsend.Conf.ReadOnce = *readOnce

	// 检查配置文件正确性
	if *check {
		_, err := logsend.LoadConfigFromFile(*config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("ok")
		os.Exit(0)
	}

	// 标准输入信息
	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	// 日志目录,有参数时取参数,取绑定参数watchDir(见watch变量说明)
	var logDirs []string
	if len(flag.Args()) > 0 {
		logDirs = flag.Args()
	} else {
		logDirs = append(logDirs, *watchDir)
	}

	// 判断是否pipe模式
	if fi.Mode()&os.ModeNamedPipe == 0 {
		logsend.WatchFiles(logDirs, *config)
	} else {
		// 按照字典顺序遍历所有已定义的flag参数
		flag.VisitAll(logsend.LoadRawConfig)
		logsend.ProcessStdin()
	}
	os.Exit(0)
}
