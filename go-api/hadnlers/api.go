package handlers

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "net/http"
    "time"
    
    "github.com/crazy1997/go-api/logging"
    "github.com/crazy1997/go-api/metrics"
)

type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    CreatedAt string `json:"created_at"`
}

type Order struct {
    ID        int       `json:"id"`
    UserID    int       `json:"user_id"`
    Total     float64   `json:"total"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

// HealthHandler возвращает статус приложения
func HealthHandler(w http.ResponseWriter, r *http.Request) {
    logging.Info("Health check requested", map[string]interface{}{
        "client_ip":  r.RemoteAddr,
        "user_agent": r.UserAgent(),
    })
    
    response := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
        "version":   "1.0.0",
        "service":   "go-api",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// UsersHandler возвращает список пользователей
func UsersHandler(w http.ResponseWriter, r *http.Request) {
    requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
    
    logging.Info("Processing users request", map[string]interface{}{
        "request_id": requestID,
        "method":     r.Method,
        "path":       r.URL.Path,
    })
    
    // 20% шанс ошибки
    if rand.Intn(100) < 20 {
        errMsg := "Database connection failed"
        logging.Error(errMsg, map[string]interface{}{
            "request_id": requestID,
            "error_type": "database_error",
            "retry_count": 2,
        })
        
        metrics.RecordError("database", "/api/users")
        http.Error(w, `{"error": "`+errMsg+`"}`, http.StatusInternalServerError)
        return
    }
    
    // Симуляция задержки БД
    delay := time.Duration(rand.Intn(200)) * time.Millisecond
    time.Sleep(delay)
    
    users := []User{
        {
            ID:        1,
            Name:      "John Doe",
            Email:     "john@example.com",
            CreatedAt: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
        },
        {
            ID:        2,
            Name:      "Jane Smith",
            Email:     "jane@example.com",
            CreatedAt: time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
        },
        {
            ID:        3,
            Name:      "Bob Johnson",
            Email:     "bob@example.com",
            CreatedAt: time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(users); err != nil {
        logging.Error("Failed to encode users response", map[string]interface{}{
            "request_id": requestID,
            "error":      err.Error(),
        })
        return
    }
    
    logging.Info("Users request completed", map[string]interface{}{
        "request_id":    requestID,
        "user_count":    len(users),
        "response_time": delay.Milliseconds(),
    })
}

// OrdersHandler создает новый заказ
func OrdersHandler(w http.ResponseWriter, r *http.Request) {
    requestID := fmt.Sprintf("order-%d", time.Now().UnixNano())
    
    if r.Method != http.MethodPost {
        logging.Warn("Invalid method for orders endpoint", map[string]interface{}{
            "request_id": requestID,
            "method":     r.Method,
            "expected":   "POST",
        })
        
        http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
        return
    }
    
    var orderData struct {
        UserID int     `json:"user_id"`
        Items  []struct {
            ProductID int `json:"product_id"`
            Quantity  int `json:"quantity"`
        } `json:"items"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&orderData); err != nil {
        logging.Error("Failed to parse order data", map[string]interface{}{
            "request_id": requestID,
            "error":      err.Error(),
        })
        
        metrics.RecordError("validation", "/api/orders")
        http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
        return
    }
    
    logging.Info("Processing order", map[string]interface{}{
        "request_id": requestID,
        "user_id":    orderData.UserID,
        "item_count": len(orderData.Items),
    })
    
    // 15% шанс ошибки обработки
    if rand.Intn(100) < 15 {
        errMsg := "Payment processing failed"
        logging.Error(errMsg, map[string]interface{}{
            "request_id": requestID,
            "error_type": "payment_error",
            "user_id":    orderData.UserID,
        })
        
        metrics.RecordError("payment", "/api/orders")
        http.Error(w, `{"error": "`+errMsg+`"}`, http.StatusPaymentRequired)
        return
    }
    
    // Симуляция обработки
    processingTime := time.Duration(rand.Intn(300)) * time.Millisecond
    time.Sleep(processingTime)
    
    order := Order{
        ID:        rand.Intn(10000),
        UserID:    orderData.UserID,
        Total:     float64(rand.Intn(1000)) + 0.99,
        Status:    "completed",
        CreatedAt: time.Now(),
    }
    
    response := map[string]interface{}{
        "success":   true,
        "order_id":  order.ID,
        "status":    order.Status,
        "total":     order.Total,
        "timestamp": order.CreatedAt.Format(time.RFC3339),
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    
    if err := json.NewEncoder(w).Encode(response); err != nil {
        logging.Error("Failed to encode order response", map[string]interface{}{
            "request_id": requestID,
            "error":      err.Error(),
        })
        return
    }
    
    // Записываем бизнес метрику
    metrics.RecordOrder()
    
    // Записываем просмотры продуктов
    for _, item := range orderData.Items {
        metrics.RecordProductView(fmt.Sprintf("%d", item.ProductID))
    }
    
    logging.Info("Order processed successfully", map[string]interface{}{
        "request_id":       requestID,
        "order_id":         order.ID,
        "processing_time":  processingTime.Milliseconds(),
        "total_amount":     order.Total,
    })
}

// ProductsHandler возвращает информацию о продуктах
func ProductsHandler(w http.ResponseWriter, r *http.Request) {
    requestID := fmt.Sprintf("prod-%d", time.Now().UnixNano())
    
    logging.Debug("Processing products request", map[string]interface{}{
        "request_id": requestID,
    })
    
    // 10% шанс медленного ответа
    if rand.Intn(100) < 10 {
        logging.Warn("Simulating slow response", map[string]interface{}{
            "request_id": requestID,
            "delay_ms":   2000,
        })
        
        time.Sleep(2 * time.Second)
    }
    
    products := []map[string]interface{}{
        {
            "id":          1,
            "name":        "Laptop Pro",
            "price":       1299.99,
            "category":    "electronics",
            "in_stock":    true,
            "rating":      4.5,
        },
        {
            "id":          2,
            "name":        "Wireless Mouse",
            "price":       49.99,
            "category":    "accessories",
            "in_stock":    true,
            "rating":      4.2,
        },
        {
            "id":          3,
            "name":        "Mechanical Keyboard",
            "price":       89.99,
            "category":    "accessories",
            "in_stock":    false,
            "rating":      4.7,
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(products); err != nil {
        logging.Error("Failed to encode products response", map[string]interface{}{
            "request_id": requestID,
            "error":      err.Error(),
        })
        return
    }
    
    logging.Info("Products request completed", map[string]interface{}{
        "request_id":   requestID,
        "product_count": len(products),
    })
}

// MetricsHandler возвращает текущие метрики приложения
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
    // Эту функцию не нужно реализовывать, так как Prometheus сам обслуживает /metrics
    // Это просто для демонстрации другого эндпоинта
    response := map[string]interface{}{
        "message": "Metrics are available at /metrics endpoint",
        "timestamp": time.Now().Unix(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}