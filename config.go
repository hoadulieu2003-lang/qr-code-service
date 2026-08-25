package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

type Config struct {
	Port          string
	Secret        string
	APIKey        string
	PublicBaseURL string
	DataFile      string
	MaxSize       int
	RecoveryLevel qrcode.RecoveryLevel
	EnableLogs    bool
	SSL           bool
}

func LoadConfig() (Config, error) {
	apiKey := getEnv("API_KEY")
	if apiKey == "" {
		return Config{}, fmt.Errorf("API_KEY must not be empty")
	}

	maxSize, err := strconv.Atoi(getEnv("MAX_SIZE", "1024"))
	if err != nil || maxSize <= 0 {
		return Config{}, fmt.Errorf("MAX_SIZE must be a positive integer")
	}

	recoveryLevels := map[string]qrcode.RecoveryLevel{
		"LOW":     qrcode.Low,
		"MEDIUM":  qrcode.Medium,
		"HIGH":    qrcode.High,
		"HIGHEST": qrcode.Highest,
	}
	recoveryName := strings.ToUpper(getEnv("RECOVERY_LEVEL", "Medium"))
	recoveryLevel, ok := recoveryLevels[recoveryName]
	if !ok {
		return Config{}, fmt.Errorf("RECOVERY_LEVEL must be Low, Medium, High, or Highest")
	}

	return Config{
		Port:          getEnv("PORT", "80"),
		Secret:        getEnv("SECRET"),
		APIKey:        apiKey,
		PublicBaseURL: strings.TrimSpace(getEnv("PUBLIC_BASE_URL")),
		DataFile:      getEnv("DATA_FILE", "data/products.json"),
		MaxSize:       maxSize,
		RecoveryLevel: recoveryLevel,
		EnableLogs:    getEnv("ENABLE_LOGS", "TRUE") == "TRUE",
		SSL:           getEnv("SSL", "FALSE") == "TRUE",
	}, nil
}

func (c Config) TraceURL(traceCode string) (string, error) {
	baseURL := strings.TrimSpace(c.PublicBaseURL)
	if baseURL == "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL is not configured")
	}
	if strings.HasSuffix(baseURL, "/") {
		return "", fmt.Errorf("PUBLIC_BASE_URL must not end with a slash")
	}

	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("PUBLIC_BASE_URL must be an absolute http or https URL")
	}

	return baseURL + "/trace/" + traceCode, nil
}

func getEnv(key string, fallback ...string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(fallback) == 0 {
			return ""
		}
		value = fallback[0]
	}

	value = strings.Trim(value, "\"")
	if normalized, isBoolean := map[string]string{
		"0":     "FALSE",
		"off":   "FALSE",
		"false": "FALSE",
		"1":     "TRUE",
		"on":    "TRUE",
		"true":  "TRUE",
	}[strings.ToLower(value)]; isBoolean {
		return normalized
	}

	return value
}
