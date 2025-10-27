package main

import (
	"log"
	"net/http"
	"runtime"
	"runtime/debug"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/internal/handler"
	"bdsw-im-ws/internal/middleware"
	"bdsw-im-ws/pkg/monitor"
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

	// 初始化WebSocket管理器
	wsManager := wsmanager.NewManager()

	// 创建Gin实例
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Auth())
	r.Use(middleware.Metrics())

	// 初始化Handler
	wsHandler := handler.NewWSHandler(wsManager, cfg)

	// 路由
	r.GET("/ws", wsHandler.HandleConnection)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":      "ok",
			"connections": wsManager.GetClientCount(),
			"goroutines":  runtime.NumGoroutine(),
		})
	})
	r.GET("/metrics", monitor.MetricsHandler())

	// 管理接口
	api := r.Group("/api")
	{
		api.GET("/stats", wsHandler.GetStats)
		api.POST("/broadcast", wsHandler.BroadcastMessage)
	}

	// 启动服务
	log.Printf("Server starting on :%s", cfg.Port)
	log.Printf("Environment: %s", cfg.Env)
	log.Printf("Runtime info - CPU: %d", runtime.NumCPU())

	log.Fatal(r.Run(":" + cfg.Port))
}
