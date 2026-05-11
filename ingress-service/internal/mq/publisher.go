package mq

import (
	"context"
	"encoding/json"
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
