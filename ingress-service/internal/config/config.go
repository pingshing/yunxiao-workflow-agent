package config

import "os"

// Config 是接入服务运行所需的环境配置。
type Config struct {
	HTTPAddr         string
	PostgresDSN      string
	RabbitMQURL      string
	RabbitMQExchange string
	CodeupToken      string
	ProjexSecret     string
	FlowToken        string
}

// Load 从环境变量读取配置，并为本地开发提供默认值。
func Load() Config {
	return Config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		PostgresDSN:      getEnv("POSTGRES_DSN", "postgres://yunxiao:yunxiao@127.0.0.1:5432/yunxiao_agent?sslmode=disable"),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),
		RabbitMQExchange: getEnv("RABBITMQ_EXCHANGE", "yunxiao.events"),
		CodeupToken:      getEnv("CODEUP_WEBHOOK_TOKEN", "dev-codeup-token"),
		ProjexSecret:     getEnv("PROJEX_WEBHOOK_SECRET", "dev-projex-secret"),
		FlowToken:        getEnv("FLOW_WEBHOOK_TOKEN", "dev-flow-token"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
