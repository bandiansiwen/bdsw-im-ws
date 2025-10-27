package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
)

type StressTester struct {
	serverURL    string
	clients      []*websocket.Conn
	mu           sync.Mutex
	connected    int32
	disconnected int32
	errors       int32
}

func NewStressTester(serverURL string) *StressTester {
	return &StressTester{
		serverURL: serverURL,
		clients:   make([]*websocket.Conn, 0),
	}
}

func (st *StressTester) ConnectClients(num int, batchSize int) {
	log.Printf("Starting to connect %d clients in batches of %d", num, batchSize)

	startTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < num; i += batchSize {
		batchStart := i
		batchEnd := i + batchSize
		if batchEnd > num {
			batchEnd = num
		}

		log.Printf("Connecting batch %d-%d", batchStart, batchEnd-1)

		for j := batchStart; j < batchEnd; j++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()

				conn, err := websocket.Dial(st.serverURL, "", "http://localhost/")
				if err != nil {
					atomic.AddInt32(&st.errors, 1)
					log.Printf("Client %d connect failed: %v", clientID, err)
					return
				}

				atomic.AddInt32(&st.connected, 1)
				st.mu.Lock()
				st.clients = append(st.clients, conn)
				st.mu.Unlock()

				// 启动心跳保持连接
				go st.keepAlive(conn, clientID)

			}(j)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // 批次间间隔
	}

	duration := time.Since(startTime)
	log.Printf("Connection phase completed in %v", duration)
	log.Printf("Successfully connected: %d/%d, Errors: %d",
		atomic.LoadInt32(&st.connected), num, atomic.LoadInt32(&st.errors))
}

func (st *StressTester) keepAlive(conn *websocket.Conn, clientID int) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	buffer := make([]byte, 256)

	for {
		select {
		case <-ticker.C:
			// 发送心跳
			message := `{"type":"ping","timestamp":` + time.Now().String() + `}`
			_, err := conn.Write([]byte(message))
			if err != nil {
				atomic.AddInt32(&st.disconnected, 1)
				log.Printf("Client %d write failed: %v", clientID, err)
				return
			}

			// 读取响应（如果有）
			_, err = conn.Read(buffer)
			if err != nil {
				atomic.AddInt32(&st.disconnected, 1)
				log.Printf("Client %d read failed: %v", clientID, err)
				return
			}
		}
	}
}

func (st *StressTester) CloseAll() {
	st.mu.Lock()
	defer st.mu.Unlock()

	for _, conn := range st.clients {
		conn.Close()
	}

	log.Printf("All connections closed. Final stats - Connected: %d, Disconnected: %d, Errors: %d",
		atomic.LoadInt32(&st.connected), atomic.LoadInt32(&st.disconnected), atomic.LoadInt32(&st.errors))
}

func main() {
	tester := NewStressTester("ws://localhost:8080/ws")

	log.Println("Starting WebSocket stress test...")

	// 测试参数
	totalConnections := 10000
	batchSize := 1000

	// 阶段1：建立连接
	log.Println("=== Phase 1: Establishing connections ===")
	tester.ConnectClients(totalConnections, batchSize)

	// 阶段2：保持连接并监控
	log.Println("=== Phase 2: Maintaining connections ===")
	log.Println("Maintaining connections for 2 minutes...")

	monitorTicker := time.NewTicker(30 * time.Second)
	defer monitorTicker.Stop()

	phase2Duration := 2 * time.Minute
	phase2End := time.Now().Add(phase2Duration)

	for time.Now().Before(phase2End) {
		select {
		case <-monitorTicker.C:
			log.Printf("Current stats - Connected: %d, Disconnected: %d, Errors: %d",
				atomic.LoadInt32(&tester.connected),
				atomic.LoadInt32(&tester.disconnected),
				atomic.LoadInt32(&tester.errors))
		}
	}

	// 阶段3：清理
	log.Println("=== Phase 3: Cleanup ===")
	tester.CloseAll()

	log.Println("Stress test completed!")
}
