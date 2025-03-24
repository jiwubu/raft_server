package raft

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// Command 表示要在状态机上执行的操作
type Command struct {
	Op    string `json:"op"`    // 操作类型：set, delete等
	Key   string `json:"key"`   // 键
	Value string `json:"value,omitempty"` // 值（可选）
}

// FSM 实现raft.FSM接口
type FSM struct {
	mu    sync.RWMutex
	data  map[string]string // 键值存储
}

// NewFSM 创建一个新的FSM
func NewFSM() *FSM {
	return &FSM{
		data: make(map[string]string),
	}
}

// Apply 将Raft日志条目应用到状态机
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Op {
	case "set":
		f.data[cmd.Key] = cmd.Value
		return nil
	case "delete":
		delete(f.data, cmd.Key)
		return nil
	default:
		return nil
	}
}

// Snapshot 返回状态机的快照
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 创建数据的副本
	data := make(map[string]string)
	for k, v := range f.data {
		data[k] = v
	}

	return &fsmSnapshot{data: data}, nil
}

// Restore 从快照恢复状态机
func (f *FSM) Restore(rc io.ReadCloser) error {
	data := make(map[string]string)
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = data
	return nil
}

// Get 返回给定键的值
func (f *FSM) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	val, ok := f.data[key]
	return val, ok
}

// fsmSnapshot 实现raft.FSMSnapshot接口
type fsmSnapshot struct {
	data map[string]string
}

// Persist 将快照写入给定的接收器
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(f.data)
	if err != nil {
		sink.Cancel()
		return err
	}

	return sink.Close()
}

// Release 释放与快照关联的资源
func (f *fsmSnapshot) Release() {}
