package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raft_server/raft"
)

// HTTPServer 是HTTP API服务器
type HTTPServer struct {
	addr  string
	store *raft.RaftStore
}

// NewHTTPServer 创建一个新的HTTP服务器
func NewHTTPServer(addr string, store *raft.RaftStore) *HTTPServer {
	return &HTTPServer{
		addr:  addr,
		store: store,
	}
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() error {
	r := gin.Default()

	// 键值存储端点
	r.GET("/kv/:key", s.handleGet)
	r.PUT("/kv/:key", s.handleSet)
	r.DELETE("/kv/:key", s.handleDelete)

	// Raft端点
	r.POST("/join", s.handleJoin)
	r.GET("/status", s.handleStatus)

	return r.Run(s.addr)
}

// handleGet 处理获取键的GET请求
func (s *HTTPServer) handleGet(c *gin.Context) {
	key := c.Param("key")
	value, ok := s.store.Get(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到键"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
}

// handleSet 处理设置键的PUT请求
func (s *HTTPServer) handleSet(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Value string `json:"value"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	if err := s.store.Set(key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleDelete 处理删除键的DELETE请求
func (s *HTTPServer) handleDelete(c *gin.Context) {
	key := c.Param("key")
	if err := s.store.Delete(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleJoin 处理加入集群的请求
func (s *HTTPServer) handleJoin(c *gin.Context) {
	var req struct {
		NodeID string `json:"node_id"`
		Addr   string `json:"addr"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	if err := s.store.Join(req.NodeID, req.Addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleStatus 返回Raft集群的状态
func (s *HTTPServer) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"leader": s.store.Leader(),
		"state":  s.store.State().String(),
	})
}
