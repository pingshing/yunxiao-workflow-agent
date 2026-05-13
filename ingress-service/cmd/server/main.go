package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"yunxiao-ingress-service/internal/config"
	"yunxiao-ingress-service/internal/httpserver"
	"yunxiao-ingress-service/internal/mq"
	"yunxiao-ingress-service/internal/repository"
)

func main() {
	// 配置
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	// 数据库
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	// MQ 生产者：启动期允许 RabbitMQ 短暂未就绪，避免容器编排抖动导致服务直接退出。
	publisher, err := mq.NewRabbitPublisherWithRetry(cfg.RabbitMQURL, cfg.RabbitMQExchange, mq.ConnectRetryOptions{
		InitialBackoff: cfg.RabbitMQConnectInitialBackoff,
		MaxBackoff:     cfg.RabbitMQConnectMaxBackoff,
		MaxWait:        cfg.RabbitMQConnectMaxWait,
	})
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer publisher.Close()

	repo := repository.NewEventRepository(db)
	server := httpserver.New(cfg, repo, publisher)

	if err := server.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
