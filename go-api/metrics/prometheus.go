package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
    "strconv"
    "time"
)

var (
    // HTTP метрики
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "Duration of HTTP requests in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
    
    httpRequestSize = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_size_bytes",
            Help:    "Size of HTTP requests in bytes",
            Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
        },
        []string{"method", "path"},
    )
    
    // Бизнес метрики
    ordersProcessed = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "orders_processed_total",
            Help: "Total number of orders processed",
        },
    )
    
    usersRegistered = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "users_registered_total",
            Help: "Total number of users registered",
        },
    )
    
    productsViewed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "products_viewed_total",
            Help: "Total number of product views",
        },
        []string{"product_id"},
    )
    
    // Ошибки
    errorCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "errors_total",
            Help: "Total number of errors",
        },
        []string{"type", "endpoint"},
    )
    
    // Системные метрики приложения
    activeRequests = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_requests",
            Help: "Number of active requests",
        },
    )
    
    responseTime95 = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "response_time_95_percentile",
            Help: "95th percentile of response time",
        },
    )
)

func Init() {
    // Регистрируем все метрики
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(httpRequestSize)
    prometheus.MustRegister(ordersProcessed)
    prometheus.MustRegister(usersRegistered)
    prometheus.MustRegister(productsViewed)
    prometheus.MustRegister(errorCounter)
    prometheus.MustRegister(activeRequests)
    prometheus.MustRegister(responseTime95)
}

func Handler() http.Handler {
    return promhttp.Handler()
}

// Middleware для сбора HTTP метрик
func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Инкрементируем активные запросы
        activeRequests.Inc()
        defer activeRequests.Dec()
        
        // Перехватываем статус код
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        // Продолжаем обработку
        next.ServeHTTP(rw, r)
        
        // Собираем метрики
        duration := time.Since(start).Seconds()
        path := r.URL.Path
        method := r.Method
        status := strconv.Itoa(rw.statusCode)
        
        httpRequestsTotal.WithLabelValues(method, path, status).Inc()
        httpRequestDuration.WithLabelValues(method, path).Observe(duration)
        
        // Размер запроса (приблизительно)
        contentLength := r.ContentLength
        if contentLength > 0 {
            httpRequestSize.WithLabelValues(method, path).Observe(float64(contentLength))
        }
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// Бизнес метрики
func RecordOrder() {
    ordersProcessed.Inc()
}

func RecordUserRegistration() {
    usersRegistered.Inc()
}

func RecordProductView(productID string) {
    productsViewed.WithLabelValues(productID).Inc()
}

func RecordError(errorType, endpoint string) {
    errorCounter.WithLabelValues(errorType, endpoint).Inc()
}

func SetResponseTime95(value float64) {
    responseTime95.Set(value)
}