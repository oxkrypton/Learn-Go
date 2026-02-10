package app

import (
    "context"
    "os"
    "os/signal"
    "syscall"
	"github.com/onexstack/fastgo/cmd/fg-apiserver/app/options"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

var configFile string

// NewFastGOCommand 创建一个 *cobra.Command 对象，用于启动应用程序.
func NewFastGOCommand() *cobra.Command {
    opts:=options.NewServerOptions()

	cmd := &cobra.Command{
		// 指定命令的名字，该名字会出现在帮助信息中
		Use: "fg-apiserver",
		// 命令的简短描述
		Short: "A very lightweight full go project",
		Long: `A very lightweight full go project, designed to help beginners quickly
        learn Go project development.`,
		// 命令出错时，不打印帮助信息。设置为 true 可以确保命令出错时一眼就能看到错误信息
		SilenceUsage: true,
		// 指定调用 cmd.Execute() 时，执行的 Run 函数
		RunE: func(cmd *cobra.Command, args []string) error {
            return run(opts)
        },
        // 设置命令运行时的参数检查，不需要指定命令行参数。例如：./fg-apiserver param1 param2
        Args: cobra.NoArgs,
    }
    
    cobra.OnInitialize(onInitialize)

    cmd.PersistentFlags().StringVarP(&configFile, "config", "c", filePath(), "Path to the fg-apiserver configuration file.")

    return cmd
}

	// run 是主运行逻辑，负责初始化日志、解析配置、校验选项并启动服务器。
func run(opts *options.ServerOptions) error {
    // 将 viper 中的配置解析到 opts.
    if err := viper.Unmarshal(opts); err != nil {
        return err
    }

    // 校验命令行选项
    if err := opts.Validate(); err != nil {
        return err
    }

    // 获取应用配置.
    // 将命令行选项和应用配置分开，可以更加灵活的处理 2 种不同类型的配置.
    cfg, err := opts.Config()
    if err != nil {
        return err
    }

    // 创建服务器实例.
    server, err := cfg.NewServer()
    if err != nil {
        return err
    }

    // 创建一个监听信号的 Context
    // 当收到 SIGINT (Ctrl+C) 或 SIGTERM 时，stop 函数会被调用，ctx.Done() 会解阻
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // 启动服务器
    return server.Run(ctx)
}