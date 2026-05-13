package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DevCodeupToken  = "dev-codeup-token"
	DevProjexSecret = "dev-projex-secret"
	DevFlowToken    = "dev-flow-token"
	DevPostgresDSN  = "postgres://yunxiao:yunxiao@127.0.0.1:5432/yunxiao_agent?sslmode=disable"
	DevRabbitMQURL  = "amqp://guest:guest@127.0.0.1:5672/"
)

// Config 是接入服务运行所需的环境配置。
type Config struct {
	AppEnv                        string
	HTTPAddr                      string
	PostgresDSN                   string
	RabbitMQURL                   string
	RabbitMQExchange              string
	RabbitMQConnectInitialBackoff time.Duration
	RabbitMQConnectMaxBackoff     time.Duration
	RabbitMQConnectMaxWait        time.Duration
	CodeupToken                   string
	ProjexSecret                  string
	FlowToken                     string
}

// Load 从环境变量读取配置，并为本地开发提供默认值。
func Load() Config {
	return Config{
		AppEnv:                        getEnv("APP_ENV", "dev"),
		HTTPAddr:                      getEnv("HTTP_ADDR", ":8080"),
		PostgresDSN:                   getEnv("POSTGRES_DSN", DevPostgresDSN),
		RabbitMQURL:                   getEnv("RABBITMQ_URL", DevRabbitMQURL),
		RabbitMQExchange:              getEnv("RABBITMQ_EXCHANGE", "yunxiao.events"),
		RabbitMQConnectInitialBackoff: getEnvDurationSeconds("RABBITMQ_CONNECT_INITIAL_BACKOFF_SECONDS", time.Second),
		RabbitMQConnectMaxBackoff:     getEnvDurationSeconds("RABBITMQ_CONNECT_MAX_BACKOFF_SECONDS", 10*time.Second),
		RabbitMQConnectMaxWait:        getEnvDurationSeconds("RABBITMQ_CONNECT_MAX_WAIT_SECONDS", 60*time.Second),
		CodeupToken:                   getEnv("CODEUP_WEBHOOK_TOKEN", DevCodeupToken),
		ProjexSecret:                  getEnv("PROJEX_WEBHOOK_SECRET", DevProjexSecret),
		FlowToken:                     getEnv("FLOW_WEBHOOK_TOKEN", DevFlowToken),
	}
}

// Validate 校验运行配置；生产环境禁止使用开发默认值或空关键配置。
func (c Config) Validate() error {
	if strings.ToLower(c.AppEnv) != "prod" {
		return nil
	}

	var problems []string
	requireNonDefault := func(name string, value string, defaultValue string) {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("%s is required", name))
			return
		}
		if value == defaultValue {
			problems = append(problems, fmt.Sprintf("%s must not use development default", name))
		}
	}

	requireNonDefault("POSTGRES_DSN", c.PostgresDSN, DevPostgresDSN)
	requireNonDefault("RABBITMQ_URL", c.RabbitMQURL, DevRabbitMQURL)
	requireNonDefault("CODEUP_WEBHOOK_TOKEN", c.CodeupToken, DevCodeupToken)
	requireNonDefault("PROJEX_WEBHOOK_SECRET", c.ProjexSecret, DevProjexSecret)
	requireNonDefault("FLOW_WEBHOOK_TOKEN", c.FlowToken, DevFlowToken)
	if strings.TrimSpace(c.RabbitMQExchange) == "" {
		problems = append(problems, "RABBITMQ_EXCHANGE is required")
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
