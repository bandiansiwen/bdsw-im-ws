package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	WebSocketConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connections_total",
			Help: "Current number of WebSocket connections",
		},
		[]string{"status"},
	)

	WebSocketMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(WebSocketConnections, WebSocketMessages)
}

func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func UpdateConnectionStats(current int, max int) {
	WebSocketConnections.WithLabelValues("current").Set(float64(current))
	WebSocketConnections.WithLabelValues("max").Set(float64(max))
}

func RecordMessageSent() {
	WebSocketMessages.WithLabelValues("sent").Inc()
}

func RecordMessageReceived() {
	WebSocketMessages.WithLabelValues("received").Inc()
}
