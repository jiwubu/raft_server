# 基于Raft的分布式键值存储系统

这是一个使用Go语言实现的分布式键值存储系统，它使用HashiCorp的Raft库来实现共识协议。

## 功能特性

- 使用Raft协议实现分布式共识
- 提供HTTP API接口与键值存储交互
- 支持集群成员变更

## 开始使用

### 前提条件

- Go 1.18或更高版本

### 构建

```bash
go mod tidy
go build -o raft-server
```

### 运行单节点

```bash
./raft-server -id node1 -http 127.0.0.1:8001 -raft 127.0.0.1:7001 -data ./data/node1 -bootstrap
```

### 运行集群

启动第一个节点：

```bash
./raft-server -id node1 -http 127.0.0.1:8001 -raft 127.0.0.1:7001 -data ./data/node1 -bootstrap
```

启动额外的节点：

```bash
./raft-server -id node2 -http 127.0.0.1:8002 -raft 127.0.0.1:7002 -data ./data/node2 -join 127.0.0.1:8001
./raft-server -id node3 -http 127.0.0.1:8003 -raft 127.0.0.1:7003 -data ./data/node3 -join 127.0.0.1:8001
```

## API使用

### 设置值

```bash
curl -X PUT http://127.0.0.1:8001/kv/mykey -d '{"value":"myvalue"}'
```

### 获取值

```bash
curl http://127.0.0.1:8001/kv/mykey
```

### 删除值

```bash
curl -X DELETE http://127.0.0.1:8001/kv/mykey
```

### 检查集群状态

```bash
curl http://127.0.0.1:8001/status
```

### 将节点加入集群

```bash
curl -X POST http://127.0.0.1:8001/join -d '{"node_id":"node4","addr":"127.0.0.1:7004"}'
```

## 系统架构

该系统由以下主要组件组成：

1. **Raft节点**：使用HashiCorp/Raft库实现的Raft共识协议节点
2. **有限状态机(FSM)**：处理和应用状态变更的组件
3. **HTTP API**：提供与系统交互的RESTful接口

## 工作原理

1. 客户端通过HTTP API发送请求到集群中的任何节点
2. 如果节点是领导者，它会处理请求；否则，它会将请求转发给领导者
3. 领导者将操作作为日志条目提交到Raft日志
4. 一旦日志条目被复制到大多数节点，它就被认为是已提交的
5. 已提交的日志条目被应用到状态机
6. 操作结果返回给客户端
