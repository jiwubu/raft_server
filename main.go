package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
	"bytes"
	"encoding/json"

	"github.com/raft_server/api"
	"github.com/raft_server/config"
	"github.com/raft_server/raft"
)

func main() {
	var serverID string
	var httpAddr string
	var raftAddr string
	var joinAddr string
	var dataDir string
	var bootstrap bool

	flag.StringVar(&serverID, "id", "", "节点ID")
	flag.StringVar(&httpAddr, "http", "127.0.0.1:8000", "HTTP API地址")
	flag.StringVar(&raftAddr, "raft", "127.0.0.1:7000", "Raft协议地址")
	flag.StringVar(&joinAddr, "join", "", "要加入的节点地址")
	flag.StringVar(&dataDir, "data", "data", "数据目录")
	flag.BoolVar(&bootstrap, "bootstrap", false, "引导集群")
	flag.Parse()

	if serverID == "" {
		log.Fatal("需要提供节点ID")
	}

	// 创建数据目录（如果不存在）
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(filepath.Join(dataDir, "raft"), 0755)

	// 初始化配置
	cfg := &config.Config{
		ServerID:  serverID,
		HTTPAddr:  httpAddr,
		RaftAddr:  raftAddr,
		JoinAddr:  joinAddr,
		DataDir:   dataDir,
		Bootstrap: bootstrap,
	}

	// 创建并启动Raft节点
	store, err := raft.NewRaftStore(cfg)
	if err != nil {
		log.Fatalf("创建Raft存储失败: %v", err)
	}

	// 启动HTTP服务器
	httpServer := api.NewHTTPServer(cfg.HTTPAddr, store)
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatalf("启动HTTP服务器失败: %v", err)
		}
	}()

	fmt.Printf("Raft节点已启动，ID: %s\n", serverID)
	fmt.Printf("HTTP服务器监听地址: %s\n", httpAddr)
	fmt.Printf("Raft服务器监听地址: %s\n", raftAddr)

	// 如果指定了加入地址，则尝试加入集群
	if cfg.JoinAddr != "" && cfg.JoinAddr != cfg.HTTPAddr {
		// 等待HTTP服务器启动
		time.Sleep(1 * time.Second)
		
		// 准备加入请求
		joinReq := struct {
			NodeID string `json:"node_id"`
			Addr   string `json:"addr"`
		}{
			NodeID: serverID,
			Addr:   raftAddr,
		}
		
		jsonData, err := json.Marshal(joinReq)
		if err != nil {
			log.Fatalf("序列化加入请求失败: %v", err)
		}
		
		// 发送加入请求
		resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Fatalf("发送加入请求失败: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("加入集群失败，状态码: %d", resp.StatusCode)
		}
		
		fmt.Printf("成功加入集群，领导者地址: %s\n", joinAddr)
	}

	// 等待中断信号
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	fmt.Println("Raft节点正在关闭...")
	store.Shutdown()
}
