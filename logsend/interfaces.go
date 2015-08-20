package logsend

// 发送者接口
type Sender interface {
	// 发送数据
	Send(interface{})
	// 设置配置参数
	SetConfig(interface{}) error
	// 名称
	Name() string
}
