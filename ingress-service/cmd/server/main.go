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

	// 数据库
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	// mq生产者
	publisher, err := mq.NewRabbitPublisher(cfg.RabbitMQURL, cfg.RabbitMQExchange)
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
