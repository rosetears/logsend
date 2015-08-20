package logsend

import (
	"github.com/ActiveState/tail"
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
)

// Using watching file in directory

// watch日志目录
func walkLogDir(dir string) (files []string, err error) {
	if string(dir[len(dir)-1]) != "/" {
		dir = dir + "/"
	}
	visit := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			Conf.Logger.Fatalln(err)
		}
		files = append(files, abs)
		return nil
	}
	//	遍历目录下的所有文件
	err = filepath.Walk(dir, visit)
	return
}

// watch文件
func WatchFiles(dirs []string, configFile string) {
	// load config 加载配置
	groups, err := LoadConfigFromFile(configFile)
	if err != nil {
		Conf.Logger.Fatalln("can't load config", err)
	}

	// get list of all files in watch dir
	files := make([]string, 0)
	for _, dir := range dirs {
		fs, err := walkLogDir(dir)
		if err != nil {
			panic(err)
		}
		for _, f := range fs {
			files = append(files, f)
		}
	}

	// assign file per group
	assignedFiles, err := assignFiles(files, groups)
	if err != nil {
		Conf.Logger.Fatalln("can't assign file per group", err)
	}

	doneCh := make(chan string)
	assignedFilesCount := len(assignedFiles)

	// 循环取日志文件记录
	for _, file := range assignedFiles {
		file.doneCh = doneCh
		go file.tail()
	}

	// 检测新文件
	if Conf.ContinueWatch {
		for _, dir := range dirs {
			go continueWatch(&dir, groups)
		}
	}

	// 等待读取完整个日志,如果是只读一次的配置,则退出
	for {
		select {
		case fpath := <-doneCh:
			assignedFilesCount = assignedFilesCount - 1
			if assignedFilesCount == 0 {
				Conf.Logger.Printf("finished reading file %+v", fpath)
				if Conf.ReadOnce {
					return
				}
			}

		}
	}

}

// 按组分配文件
func assignFiles(files []string, groups []*Group) (outFiles []*File, err error) {
	for _, group := range groups {
		var assignedFiles []*File
		if assignedFiles, err = getFilesByGroup(files, group); err == nil {
			for _, assignedFile := range assignedFiles {
				outFiles = append(outFiles, assignedFile)
			}
		} else {
			return
		}
	}
	return
}

// 按组查询文件
func getFilesByGroup(allFiles []string, group *Group) ([]*File, error) {
	files := make([]*File, 0)
	// 当期组的mask
	regex := *group.Mask
	// 遍历所有文件与组mask匹配
	for _, f := range allFiles {
		if !regex.MatchString(filepath.Base(f)) {
			continue
		}
		// 根据日志f创建File
		file, err := NewFile(f)
		if err != nil {
			return files, err
		}
		file.group = group
		files = append(files, file)
	}
	return files, nil
}

// 检测新创建的文件
func continueWatch(dir *string, groups []*Group) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Conf.Logger.Fatal(err)
	}

	done := make(chan bool)

	// Process events
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsCreate() {
					files := make([]string, 0)
					file, err := filepath.Abs(ev.Name)
					if err != nil {
						Conf.Logger.Printf("can't get file %+v", err)
						continue
					}
					files = append(files, file)
					assignFiles(files, groups)
				}
			case err := <-watcher.Error:
				Conf.Logger.Println("error:", err)
			}
		}
	}()

	err = watcher.Watch(*dir)
	if err != nil {
		Conf.Logger.Fatal(err)
	}

	<-done

	/* ... do stuff ... */
	watcher.Close()
}

// 创建File对象
func NewFile(fpath string) (*File, error) {
	file := &File{}
	var err error
	// 判断读取方式
	if Conf.ReadWholeLog && Conf.ReadOnce {
		// 读整个文件并且读一次
		Conf.Logger.Printf("read whole file once %+v", fpath)
		file.Tail, err = tail.TailFile(fpath, tail.Config{})
	} else if Conf.ReadWholeLog {
		// 读整个文件和增量内容
		Conf.Logger.Printf("read whole file and continue %+v", fpath)
		file.Tail, err = tail.TailFile(fpath, tail.Config{Follow: true, ReOpen: true})
	} else {
		// 从文件末尾开始读取
		// offset 文件指针的位置
		// whence 相对位置标识: 0代表相对文件开始的位置,1代表相对当前位置,2代表相对文件结尾的位置
		seekInfo := &tail.SeekInfo{Offset: 0, Whence: 2}
		file.Tail, err = tail.TailFile(fpath, tail.Config{Follow: true, ReOpen: true, Location: seekInfo})
	}
	return file, err
}

// 文件类型
type File struct {
	Tail   *tail.Tail
	group  *Group
	doneCh chan string
}

// 读取文件尾行
func (self *File) tail() {
	Conf.Logger.Printf("start tailing %+v", self.Tail.Filename)
	defer func() { self.doneCh <- self.Tail.Filename }()
	for line := range self.Tail.Lines {
		checkLineRules(&line.Text, self.group.Rules)
	}
}

// 对行数据进行规则匹配,匹配后发送
func checkLineRule(line *string, rule *Rule) {
	match := rule.Match(line)
	if match != nil {
		rule.send(match)
	}
}

// 对行数据进行规则检查
func checkLineRules(line *string, rules []*Rule) {
	for _, rule := range rules {
		checkLineRule(line, rule)
	}
}
