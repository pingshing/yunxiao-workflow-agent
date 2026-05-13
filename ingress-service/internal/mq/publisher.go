package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"yunxiao-ingress-service/internal/model"
)

// Publisher 定义标准事件发布接口，避免业务代码直接绑定 RabbitMQ 实现。
type Publisher interface {
	Publish(ctx context.Context, event model.NormalizedEvent) error
}

// RabbitPublisher 使用 RabbitMQ topic exchange 发布标准事件。
type RabbitPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

var newRabbitPublisher = NewRabbitPublisher

// ConnectRetryOptions 定义 RabbitMQ 启动连接重试策略。
type ConnectRetryOptions struct {
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	MaxWait        time.Duration
	Sleep          func(time.Duration)
}

// NewRabbitPublisher 创建 RabbitMQ 发布器，并确保目标 exchange 存在。
func NewRabbitPublisher(url string, exchange string) (*RabbitPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitPublisher{conn: conn, channel: channel, exchange: exchange}, nil
}

// NewRabbitPublisherWithRetry 创建 RabbitMQ 发布器，并在启动期连接失败时按指数退避重试。
func NewRabbitPublisherWithRetry(url string, exchange string, options ConnectRetryOptions) (*RabbitPublisher, error) {
	initialBackoff := positiveDuration(options.InitialBackoff, time.Second)
	maxBackoff := positiveDuration(options.MaxBackoff, 10*time.Second)
	maxWait := positiveDuration(options.MaxWait, 60*time.Second)
	sleep := options.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	backoff := initialBackoff
	waited := time.Duration(0)
	var lastErr error
	for attempt := 1; ; attempt++ {
		publisher, err := newRabbitPublisher(url, exchange)
		if err == nil {
			if attempt > 1 {
				log.Printf("rabbitmq connected after retry attempts=%d", attempt)
			}
			return publisher, nil
		}
		lastErr = err
		if waited+backoff >= maxWait {
			break
		}
		log.Printf("rabbitmq connect failed attempt=%d backoff=%s error=%q", attempt, backoff, err.Error())
		sleep(backoff)
		waited += backoff
		backoff = nextBackoff(backoff, maxBackoff)
	}
	return nil, fmt.Errorf("connect rabbitmq after %s: %w", maxWait, lastErr)
}

// Publish 使用 event_type 作为 routing key 发布标准事件。
func (p *RabbitPublisher) Publish(ctx context.Context, event model.NormalizedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		event.EventType,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			MessageId:    event.EventID,
			Headers: amqp.Table{
				"x-trace-id":   event.TraceID,
				"x-event-id":   event.EventID,
				"x-event-type": event.EventType,
				"x-source":     event.Source,
			},
			Body: body,
		},
	)
}

// Close 关闭 RabbitMQ channel 和 connection。
func (p *RabbitPublisher) Close() error {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func positiveDuration(value time.Duration, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func nextBackoff(current time.Duration, maxBackoff time.Duration) time.Duration {
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}
