package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/internal/handler"
	"bdsw-im-ws/internal/middleware"
	"bdsw-im-ws/pkg/monitor"
	"bdsw-im-ws/pkg/service_factory"
	"bdsw-im-ws/pkg/wsmanager"

	"github.com/gin-gonic/gin"
)

func setupSystemOptimization() {
	// 1. 设置GOMAXPROCS - 利用多核
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	log.Printf("Set GOMAXPROCS to: %d", numCPU)

	// 2. 优化GC - 在内存敏感场景下更积极的GC
	debug.SetGCPercent(20)
	log.Printf("Set GC percent to: 20")

	// 3. 设置阻塞分析阈值
	runtime.SetBlockProfileRate(1)

	// 4. 设置Mutex分析
	runtime.SetMutexProfileFraction(1)
}

func main() {
	// 系统优化
	setupSystemOptimization()

	// 启动pprof监控（在另一个端口）
	go func() {
		log.Println("Starting pprof server on :6060")
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	// 加载配置
	cfg := config.Load()

	// 设置Gin模式
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化服务工厂
	serviceFactory, err := service_factory.NewServiceFactory(cfg)
	if err != nil {
		log.Fatalf("Failed to create service factory: %v", err)
	}
	defer serviceFactory.Close()

	// 初始化服务客户端
	if err := serviceFactory.Init(); err != nil {
		log.Printf("Warning: Failed to init some services: %v", err)
	}

	// 初始化WebSocket管理器
	wsManager := wsmanager.NewManager()

	// 创建Gin实例
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Metrics())

	// 初始化Handler
	wsHandler := handler.NewWSHandler(wsManager, cfg, serviceFactory)

	// 路由
	r.GET("/ws", wsHandler.HandleConnection)
	r.GET("/health", func(c *gin.Context) {
		healthInfo := gin.H{
			"status":      "ok",
			"connections": wsManager.GetClientCount(),
			"goroutines":  runtime.NumGoroutine(),
			"timestamp":   time.Now(),
		}

		// 添加服务可用性信息
		services := gin.H{}
		if serviceFactory.HasAuthService() {
			services["auth_service"] = "available"
		} else {
			services["auth_service"] = "unavailable"
		}
		if serviceFactory.HasBusinessService() {
			services["business_service"] = "available"
		} else {
			services["business_service"] = "unavailable"
		}
		healthInfo["services"] = services

		c.JSON(200, healthInfo)
	})
	r.GET("/metrics", monitor.MetricsHandler())

	// 管理接口
	api := r.Group("/api")
	{
		api.GET("/stats", wsHandler.GetStats)
		api.GET("/users/online", wsHandler.GetOnlineUsers)
		api.POST("/broadcast", wsHandler.BroadcastMessage)
	}

	// 启动服务
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// 优雅关闭
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Printf("Server started on :%s", cfg.Port)
	log.Printf("Environment: %s", cfg.Env)
	log.Printf("Runtime info - CPU: %d", runtime.NumCPU())

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
