This is a very important context, plz remember it
## Go 版黑马点评：项目实时上下文 (Context Anchor)

### 1. 核心背景与开发环境

- **项目目标**：参考 Java 版“黑马点评”教程，使用 Go 语言进行高性能、工程化的业务复现。
- **系统环境**：全程基于 **macOS** 开发，不接受 Windows 相关技术建议。

### 2. 当前技术栈选型

| **模块**        | **选型**          | **状态/说明**                                         |
| ------------- | --------------- | ------------------------------------------------- |
| **Web 框架**    | **Gin**         | 已成功运行并实现 `/ping` 测试接口。                            |
| **Redis 客户端** | **go-redis/v9** | 已实现连接池，通过 Viper 动态读取配置并完成通讯。                      |
| **数据库 ORM**   | **GORM**        | 选定用于 MySQL 交互，待正式接入业务模型。                          |
| **配置管理**      | **Viper**       | **已上线**。支持从 `config.yaml` 映射到 `GlobalConfig` 结构体。 |
### 3. 架构与开发约定

- **分层模型**：严格遵循 `Controller -> Service -> Repository` 结构。
    
- **数据交换**：统一返回格式为 `{ "code": int, "msg": string, "data": any }`。
    
- **并发控制**：全链路显式传递 `context.Context`。
    
- **规范化要求**：
	- 提出需求后在正常回复后列出实现该需求涉及到的改动/新增文件
	- 按照代码逻辑提示用户下一个应该修改/创建哪个文件
	- 等待用户发送目标文件一个一个进行修改
	- 新增/改动代码需要添加注释

	- (IMPORTANT)若信息不足、代码逻辑模糊请向用户索要信息，切记不可胡编乱造
    
    - **错误处理**：业务逻辑严禁 `panic`，必须显式返回 `error`；初始化阶段（如 DB/Redis 失败）允许 `panic` 终止。
        
### 4. 目录结构现状

Plaintext

```
Go-redis/
├── cmd/
│   └── main.go              # 程序入口（
├── config/
│   └── config.yaml          # 外部配置文件
├── internal/
│   ├── config/
│   │   └── config.go        # Viper 配置解析与 GlobalConfig 定义
│   └── repository/
│       └── redis.go         # Redis 客户端逻辑
├── go.mod                   # module go-redis
└── go.sum
```
