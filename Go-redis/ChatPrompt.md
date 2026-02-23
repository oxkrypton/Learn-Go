Go 版黑马点评：项目实时上下文 (Context Anchor) v1.21. 核心背景与开发环境项目目标：参考 Java 版“黑马点评”教程，使用 Go 语言进行高性能、工程化的业务复现（练习项目，认真对待）。系统环境：全程基于 macOS 开发，不接受 Windows 相关技术建议。2. 当前技术栈选型模块选型状态/说明Web 框架Gin已成功运行并实现 /ping 测试接口，运行在 8080 端口。Redis 客户端go-redis/v9已实现连接池，支持密码校验与 Ping 强检查。数据库 ORMGORM已上线。接入 MySQL 8.0，配置了连接池与 Ping 校验。配置管理Viper已上线。支持从 config.yaml 映射到 GlobalConfig 结构体。反向代理Nginx已上线。负责静态页面分发（8081 端口）并将 /api/ 路由反向代理至 Gin 后端（host.docker.internal:8080）。基础设施Docker Compose一键启动 MySQL、Redis 与 Nginx，支持 hmdp.sql 自动初始化。3. 架构与开发约定分层模型：严格遵循 Controller -> Service -> Repository 结构。数据交换：统一返回格式为 { "code": int, "msg": string, "data": any }。并发控制：全链路显式传递 context.Context。规范化要求：(IMPORTANT) 若信息不足、代码逻辑模糊请向用户索要信息，严禁胡编乱造。提出需求后，列出涉及到的改动/新增文件。按照代码逻辑提示用户下一个应该修改/创建哪个文件。等待用户发送目标文件一个一个进行修改。新增/改动代码需要添加注释。错误处理：业务逻辑严禁 panic，必须显式返回 error；初始化阶段（如 DB/Redis 失败）允许 panic 终止。4. 目录结构现状 (Updated)PlaintextGo-redis/
├── cmd/
│   └── main.go              
├── config/
│   └── config.yaml          # 配置参数
├── internal/
│   ├── config/
│   │   └── config.go        # Viper 映射逻辑
│   ├── controller/          # [待完善] 接收 HTTP 请求
│   ├── dto/                 
│   │   └── dto.go           # 统一 Result 响应与 UserDTO
│   ├── model/               # 数据库实体 (PO)
│   │   ├── blog.go, follow.go, shop.go, user.go, voucher.go
│   │   └── shop_type.go     # [新增] 商铺类型实体
│   ├── pkg/database/        
│   │   ├── mysql.go         # GORM 初始化与连接池
│   │   └── redis.go         # go-redis 连接池
│   ├── repository/          # 数据访问层 (CRUD接口化)
│   │   ├── blog_repository.go
│   │   ├── follow_repository.go
│   │   ├── shop_repository.go
│   │   ├── user_repository.go
│   │   └── voucher_repository.go
│   └── service/             # [待完善] 核心业务逻辑层
├── nginx/                   # [新增] Nginx 前端与代理配置
│   ├── nginx.conf           # 反向代理配置文件
│   └── html/
│       └── hmdp/            # 前端静态资源
├── hmdp.sql                 # 核心数据库脚本
└── docker-compose.yaml      # 一键基础设施启动 (已包含 Nginx)