package logging

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "runtime"
    "sync"
    "time"
)

// ELKLogger отправляет логи напрямую в Logstash
type ELKLogger struct {
    logstashURL string
    httpClient  *http.Client
    serviceName string
    environment string
    hostname    string
    serverIP    string
    mu          sync.Mutex
}

var (
    loggerInstance *ELKLogger
    once           sync.Once
)

type LogEntry struct {
    Timestamp   string                 `json:"@timestamp"`
    Level       string                 `json:"level"`
    Service     string                 `json:"service"`
    Message     string                 `json:"message"`
    Fields      map[string]interface{} `json:"fields,omitempty"`
    Environment string                 `json:"environment"`
    Host        string                 `json:"host"`
    ServerIP    string                 `json:"server_ip"`
    GoVersion   string                 `json:"go_version"`
}

func InitLogger() *ELKLogger {
    once.Do(func() {
        hostname, _ := os.Hostname()
        
        // Получаем внешний IP сервера
        serverIP := os.Getenv("SERVER_IP")
        if serverIP == "" {
            serverIP = "147.45.183.143" // Ваш IP сервера
        }
        
        // Logstash URL - используем localhost внутри контейнера
        logstashURL := "http://logstash:5000"
        
        loggerInstance = &ELKLogger{
            logstashURL: logstashURL,
            httpClient: &http.Client{
                Timeout: 5 * time.Second,
                Transport: &http.Transport{
                    MaxIdleConns:        100,
                    MaxIdleConnsPerHost: 100,
                    IdleConnTimeout:     90 * time.Second,
                },
            },
            serviceName: "go-api",
            environment: os.Getenv("ENVIRONMENT"),
            hostname:    hostname,
            serverIP:    serverIP,
        }
        
        if loggerInstance.environment == "" {
            loggerInstance.environment = "production"
        }
        
        // Тестовое сообщение при инициализации
        loggerInstance.Log("INFO", "Logger initialized on production server", map[string]interface{}{
            "server_ip":     serverIP,
            "logstash_url":  logstashURL,
            "environment":   loggerInstance.environment,
            "hostname":      hostname,
        })
    })
    
    return loggerInstance
}


func GetLogger() *ELKLogger {
    if loggerInstance == nil {
        panic("Logger not initialized. Call InitLogger first")
    }
    return loggerInstance
}

func (l *ELKLogger) Log(level, message string, fields map[string]interface{}) {
    go l.sendLogAsync(level, message, fields)
    
    // Также выводим в консоль для отладки
    l.logToConsole(level, message, fields)
}

func (l *ELKLogger) sendLogAsync(level, message string, fields map[string]interface{}) {
    entry := l.createLogEntry(level, message, fields)
    
    jsonData, err := json.Marshal(entry)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to marshal log: %v\n", err)
        return
    }
    
    req, err := http.NewRequest("POST", l.logstashURL, bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create log request: %v\n", err)
        return
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := l.httpClient.Do(req)
    if err != nil {
        // В случае ошибки пишем в stderr
        fmt.Fprintf(os.Stderr, "Failed to send log to ELK: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        fmt.Fprintf(os.Stderr, "Logstash returned error: %d\n", resp.StatusCode)
    }
}

func (l *ELKLogger) createLogEntry(level, message string, fields map[string]interface{}) LogEntry {
    if fields == nil {
        fields = make(map[string]interface{})
    }
    
    // Добавляем информацию о вызове
    _, file, line, ok := runtime.Caller(3)
    if ok {
        fields["caller"] = fmt.Sprintf("%s:%d", file, line)
    }
    
    return LogEntry{
        Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
        Level:       level,
        Service:     l.serviceName,
        Message:     message,
        Fields:      fields,
        Environment: l.environment,
        Host:        l.hostname,
        GoVersion:   runtime.Version(),
    }
}

func (l *ELKLogger) logToConsole(level, message string, fields map[string]interface{}) {
    color := "\033[0m"
    switch level {
    case "ERROR":
        color = "\033[31m" // Красный
    case "WARN":
        color = "\033[33m" // Желтый
    case "INFO":
        color = "\033[32m" // Зеленый
    case "DEBUG":
        color = "\033[36m" // Голубой
    }
    
    timestamp := time.Now().Format("15:04:05.000")
    fmt.Printf("%s[%s] %-5s %s\033[0m", color, timestamp, level, message)
    
    if len(fields) > 0 {
        fmt.Print(" | ")
        for k, v := range fields {
            fmt.Printf("%s=%v ", k, v)
        }
    }
    fmt.Println()
}

// Удобные методы
func (l *ELKLogger) Info(message string, fields map[string]interface{}) {
    l.Log("INFO", message, fields)
}

func (l *ELKLogger) Error(message string, fields map[string]interface{}) {
    l.Log("ERROR", message, fields)
}

func (l *ELKLogger) Warn(message string, fields map[string]interface{}) {
    l.Log("WARN", message, fields)
}

func (l *ELKLogger) Debug(message string, fields map[string]interface{}) {
    if l.environment == "development" {
        l.Log("DEBUG", message, fields)
    }
}

// Глобальные функции для удобства
func Info(message string, fields map[string]interface{}) {
    GetLogger().Info(message, fields)
}

func Error(message string, fields map[string]interface{}) {
    GetLogger().Error(message, fields)
}

func Warn(message string, fields map[string]interface{}) {
    GetLogger().Warn(message, fields)
}

func Debug(message string, fields map[string]interface{}) {
    GetLogger().Debug(message, fields)
}