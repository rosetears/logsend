package logsend

import (
	"bytes"
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"text/template"
)

// DOCS
// https://github.com/go-sql-driver/mysql#dsn-data-source-name
var (
	mysqlCh    = make(chan *string, 0)
	mysqlHost  = flag.String("mysql-host", "", "Example: user:password@/database?timeout=30s&strict=true")
	mysqlQuery = flag.String("mysql-query", "", "Example: insert into test1(teststring, testfloat) values('{{.gate}}', {{.exec_time}});")
)

// 初始化注册发送方
func init() {
	RegisterNewSender("mysql", InitMysql, NewMysqlSender)
}

// 初始化数据库连接,
func InitMysql(conf interface{}) {
	host := conf.(map[string]interface{})["host"].(string)
	db, err := sql.Open("mysql", host)
	if err != nil {
		panic(err.Error())
	}

	// 启动goroutine处理数据,通过mysqlCh交换数据
	go func() {
		defer db.Close()
		Conf.Logger.Println("mysql queue is starts")
		for query := range mysqlCh {

			err = db.Ping()
			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}

			debug("mysql exec query: ", *query)
			// TODO: exec query with transaction support
			if Conf.DryRun {
				continue
			}

			trx, err := db.Begin()
			if err != nil {
				Conf.Logger.Println("mysql init transaction ", err, *query)
			}

			for _, q := range strings.Split(*query, ";") {
				if q == "" {
					continue
				}
				if _, err = db.Exec(q + ";"); err != nil {
					break
				}

			}
			if err != nil {
				trx.Rollback()
				Conf.Logger.Println("rollback ", err, *query)
				continue
			}
			trx.Commit()
		}
	}()
	return
}

// 创建新的MysqlSender
func NewMysqlSender() Sender {
	mysqlSender := &MysqlSender{}
	mysqlSender.sendCh = mysqlCh
	return Sender(mysqlSender)
}

// Mysql发送者结构
type MysqlSender struct {
	sendCh chan *string
	tmpl   *template.Template
}

// 名称
func (self *MysqlSender) Name() string {
	return "mysql"
}

// 设置配置
func (self *MysqlSender) SetConfig(rawConfig interface{}) error {
	var query string
	switch rawConfig.(map[string]interface{})["query"].(type) {
	case []interface{}:
		for _, s := range rawConfig.(map[string]interface{})["query"].([]interface{}) {
			query = query + s.(string)
		}
	case string:
		query = rawConfig.(map[string]interface{})["query"].(string)
	}
	self.tmpl, _ = template.New("query").Parse(query)
	debug("set config to mysql sender")
	return nil
}

// 发送数据
func (self *MysqlSender) Send(data interface{}) {
	buf := new(bytes.Buffer)
	// 根据模板转换数据
	err := self.tmpl.Execute(buf, data)
	if err != nil {
		Conf.Logger.Println("mysql template error ", err, data)
	}
	str := buf.String()
	// 数据写入channel
	self.sendCh <- &str
	return
}
