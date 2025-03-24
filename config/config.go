package config

// Config 保存Raft服务器的配置
type Config struct {
	ServerID  string // 服务器ID
	HTTPAddr  string // HTTP API地址
	RaftAddr  string // Raft协议地址
	JoinAddr  string // 要加入的节点地址
	DataDir   string // 数据目录
	Bootstrap bool   // 是否引导集群
}
