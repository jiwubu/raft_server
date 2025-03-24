package raft

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/raft_server/config"
)

// RaftStore 封装Raft服务器和FSM
type RaftStore struct {
	config *config.Config
	raft   *raft.Raft
	fsm    *FSM
}

// NewRaftStore 创建一个新的Raft存储
func NewRaftStore(config *config.Config) (*RaftStore, error) {
	// 创建FSM
	fsm := NewFSM()

	// Raft配置
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(config.ServerID)
	raftConfig.SnapshotInterval = 20 * time.Second
	raftConfig.SnapshotThreshold = 1024

	// 创建Raft传输
	addr, err := net.ResolveTCPAddr("tcp", config.RaftAddr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(config.RaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// 创建日志存储和稳定存储
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(config.DataDir, "raft", "raft-log.bolt"))
	if err != nil {
		return nil, err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(config.DataDir, "raft", "raft-stable.bolt"))
	if err != nil {
		return nil, err
	}

	// 创建快照存储
	snapshotStore, err := raft.NewFileSnapshotStore(filepath.Join(config.DataDir, "raft"), 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	// 创建Raft实例
	r, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	// 如果需要，引导集群
	if config.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(config.ServerID),
					Address: transport.LocalAddr(),
				},
			},
		}
		r.BootstrapCluster(configuration)
	}

	return &RaftStore{
		config: config,
		raft:   r,
		fsm:    fsm,
	}, nil
}

// Get 从存储中获取值
func (s *RaftStore) Get(key string) (string, bool) {
	return s.fsm.Get(key)
}

// Set 在存储中设置值
func (s *RaftStore) Set(key, value string) error {
	cmd := &Command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	return s.applyCommand(cmd)
}

// Delete 从存储中删除值
func (s *RaftStore) Delete(key string) error {
	cmd := &Command{
		Op:  "delete",
		Key: key,
	}
	return s.applyCommand(cmd)
}

// Join 将节点加入Raft集群
func (s *RaftStore) Join(nodeID, addr string) error {
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	// 检查服务器是否已经在集群中
	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// 已经加入
			return nil
		}
	}

	// 将服务器添加到集群
	addFuture := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}

	return nil
}

// Shutdown 关闭Raft服务器
func (s *RaftStore) Shutdown() {
	s.raft.Shutdown().Error()
}

// Leader 返回当前的领导者
func (s *RaftStore) Leader() string {
	return string(s.raft.Leader())
}

// State 返回当前的Raft状态
func (s *RaftStore) State() raft.RaftState {
	return s.raft.State()
}

// applyCommand 将命令应用到Raft日志
func (s *RaftStore) applyCommand(cmd *Command) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	// 将命令应用到Raft日志
	future := s.raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		return err
	}

	// 检查应用是否返回错误
	if err, ok := future.Response().(error); ok && err != nil {
		return err
	}

	return nil
}
