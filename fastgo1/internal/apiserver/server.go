package apiserver

import (
    "fmt"
    "context"
    "time"

    genericoptions "github.com/onexstack/fastgo/pkg/options"
)

// Config 配置结构体，用于存储应用相关的配置.
// 不用 viper.Get，是因为这种方式能更加清晰的知道应用提供了哪些配置项.
type Config struct {
    MySQLOptions *genericoptions.MySQLOptions
}

// Server 定义一个服务器结构体类型.
type Server struct {
    cfg *Config
}

// NewServer 根据配置创建服务器.
func (cfg *Config) NewServer() (*Server, error) {
    return &Server{cfg: cfg}, nil
}

// Run 运行应用.
func (s *Server) Run(ctx context.Context) error {
    fmt.Println("服务已启动：")
    fmt.Printf("Read MySQL host from config: %s\n", s.cfg.MySQLOptions.Addr)

    <-ctx.Done()

    return s.Close()
    //select {} // 调用 select 语句，阻塞防止进程退出
}

func (s *Server) Close() error {
    fmt.Println("正在清理资源并关闭数据库连接...")
    time.Sleep(1 * time.Second) // 模拟清理耗时
    fmt.Println("服务器已安全退出")
    return nil
}