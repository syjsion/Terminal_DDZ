package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/syjsion/Terminal_DDZ/internal/config"
	"github.com/syjsion/Terminal_DDZ/internal/logging"
	"github.com/syjsion/Terminal_DDZ/internal/tui"
	"github.com/syjsion/Terminal_DDZ/internal/version"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("terminal-ddz", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "配置文件路径（默认 ./config.toml）")
	debug := flags.Bool("debug", false, "将脱敏调试日志写入 ./terminal-ddz.log")
	showVersion := flags.Bool("version", false, "显示版本信息")
	seedFlag := flags.Int64("seed", 0, "指定洗牌随机种子（0 表示随机）")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Terminal 斗地主")
		fmt.Fprintln(stderr, "用法: terminal-ddz [选项]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "未知参数: %s\n", flags.Arg(0))
		flags.Usage()
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return 0
	}
	seed := *seedFlag
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	cfg, usedPath, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var debugLogger *log.Logger
	var logFile *os.File
	if *debug {
		logFile, err = os.OpenFile("terminal-ddz.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			fmt.Fprintf(stderr, "创建调试日志失败: %v\n", err)
			return 1
		}
		defer logFile.Close()
		var secrets []string
		for _, provider := range cfg.Providers {
			if provider.APIKey != "" {
				secrets = append(secrets, provider.APIKey)
			}
		}
		writer := &logging.RedactingWriter{Writer: logFile, Secrets: secrets}
		debugLogger = log.New(writer, "terminal-ddz ", log.LstdFlags|log.Lmicroseconds)
		debugLogger.Printf("启动 version=%s config=%q seed=%d", version.Version, usedPath, seed)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := tui.Run(ctx, cfg, seed, debugLogger); err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return 0
		}
		fmt.Fprintf(stderr, "启动 TUI 失败: %v\n", err)
		return 1
	}
	return 0
}
