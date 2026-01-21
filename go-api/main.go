package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	handlers "github.com/crazy1997/go-api/hadnlers"
	"github.com/crazy1997/go-api/logging"
	"github.com/crazy1997/go-api/metrics"
	"github.com/gorilla/mux"
)

func main() {
	// Инициализация логгера
	logger := logging.InitLogger()

	// Инициализация метрик
	metrics.Init()

	// Создаем роутер
	r := mux.NewRouter()

	// Глобальный middleware для метрик
	r.Use(metrics.MetricsMiddleware)

	// API эндпоинты
	r.HandleFunc("/api/health", handlers.HealthHandler).Methods("GET")
	r.HandleFunc("/api/users", handlers.UsersHandler).Methods("GET")
	r.HandleFunc("/api/orders", handlers.OrdersHandler).Methods("POST")
	r.HandleFunc("/api/products", handlers.ProductsHandler).Methods("GET")
	r.HandleFunc("/api/metrics/info", handlers.MetricsHandler).Methods("GET")

	// Prometheus метрики
	r.Handle("/metrics", metrics.Handler())

	// Статика
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	// Настройка сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info(fmt.Sprintf("Starting server on %s:%s", "0.0.0.0", port), map[string]interface{}{
			"environment": os.Getenv("ENVIRONMENT"),
			"server_ip":   "147.45.183.143",
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Ожидаем сигнал остановки
	<-stop

	logger.Info("Shutting down server...", nil)

	// Даем время на завершение обработки запросов
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info("Server stopped gracefully", nil)
}
