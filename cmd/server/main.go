package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/pkg/area"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"github.com/wxlbd/ruoyi-mall-go/pkg/logger"

	"go.uber.org/zap"
)

// shutdownTimeout 优雅退出时等待在途请求结束的时间
const shutdownTimeout = 15 * time.Second

func main() {
	// 1. 初始化配置（含启动期校验，缺关键配置直接失败）
	if err := config.Load(); err != nil {
		panic(err)
	}

	// 2. 初始化日志
	logger.Init()
	logger.Info("Config and Logger initialized", zap.String("env", config.C.App.Env))

	// 3. 应用认证密钥
	// JWT 使用已验证的 security.jwt_secret。

	// 4. 初始化地区数据
	if err := area.Init("configs/area.csv"); err != nil {
		logger.Log.Warn("Failed to init area data", zap.Error(err))
	}

	// 5. 初始化应用 (通过 Wire 注入)
	engine, err := InitApp()
	if err != nil {
		logger.Log.Fatal("failed to init app", zap.Error(err))
	}

	// 6. 启动服务
	srv := &http.Server{Addr: config.C.HTTP.Port, Handler: engine}
	go func() {
		logger.Info("Server starting...", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// 7. 优雅退出：先停止接收新请求，再等待在途请求结束。
	// 未完成的支付通知任务由 pay_notify_task 表持久化，重启后由定时任务继续。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server forced to shutdown", zap.Error(err))
	}
	logger.Info("Server exited")
}
