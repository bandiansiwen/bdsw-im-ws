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

	"github.com/bdsw/bdsw-im-ws/internal/redis"
	"github.com/bdsw/bdsw-im-ws/internal/registry"

	"github.com/bdsw/bdsw-im-ws/internal/config"
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

	// 加载应用配置
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Failed to load app config: %v", err)
	}

	appConfig := config.GetConfig()
	log.Printf("Starting IMA Gateway in %s environment", appConfig.App.Environment)

	// 初始化 Redis 客户端
	redisConfig := appConfig.Redis.Standalone
	redisClient := redis.NewClient(&redis.Config{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	if err := redisClient.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Redis connected successfully")

	// 创建服务注册器（解决循环导入）
	serviceRegistry, err := registry.NewServiceRegistry(redisClient)
	if err != nil {
		log.Fatalf("Failed to create service registry: %v", err)
	}

	// 注册 Dubbo 服务
	serviceRegistry.RegisterDubboServices()

	// 启动所有服务
	if err := serviceRegistry.StartAll(); err != nil {
		log.Fatalf("Failed to start IMA Gateway: %v", err)
	}

	log.Println("IMA Gateway started successfully")

	// 7. 等待中断信号
	waitForShutdown(serviceRegistry)
}

func waitForShutdown(registry *registry.ServiceRegistry) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	<-signals
	log.Println("Received shutdown signal, gracefully shutting down...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := registry.ShutdownAll(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("IMA Gateway shutdown completed")
}
