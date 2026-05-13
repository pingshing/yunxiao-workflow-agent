package mq

import (
	"errors"
	"testing"
	"time"
)

func TestNewRabbitPublisherWithRetryRetriesUntilSuccess(t *testing.T) {
	previous := newRabbitPublisher
	defer func() { newRabbitPublisher = previous }()

	attempts := 0
	var slept []time.Duration
	newRabbitPublisher = func(url string, exchange string) (*RabbitPublisher, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("rabbitmq unavailable")
		}
		return &RabbitPublisher{exchange: exchange}, nil
	}

	publisher, err := NewRabbitPublisherWithRetry("amqp://example", "yunxiao.events", ConnectRetryOptions{
		InitialBackoff: time.Second,
		MaxBackoff:     5 * time.Second,
		MaxWait:        time.Minute,
		Sleep: func(duration time.Duration) {
			slept = append(slept, duration)
		},
	})
	if err != nil {
		t.Fatalf("NewRabbitPublisherWithRetry() error = %v", err)
	}
	if publisher == nil {
		t.Fatal("publisher is nil")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(slept) != 2 || slept[0] != time.Second || slept[1] != 2*time.Second {
		t.Fatalf("slept = %v, want [1s 2s]", slept)
	}
}

func TestNewRabbitPublisherWithRetryReturnsLastErrorAfterMaxWait(t *testing.T) {
	previous := newRabbitPublisher
	defer func() { newRabbitPublisher = previous }()

	attempts := 0
	newRabbitPublisher = func(url string, exchange string) (*RabbitPublisher, error) {
		attempts++
		return nil, errors.New("rabbitmq unavailable")
	}

	_, err := NewRabbitPublisherWithRetry("amqp://example", "yunxiao.events", ConnectRetryOptions{
		InitialBackoff: time.Second,
		MaxBackoff:     time.Second,
		MaxWait:        time.Second,
		Sleep:          func(time.Duration) {},
	})
	if err == nil {
		t.Fatal("error is nil, want failure")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestNextBackoffCapsAtMaxBackoff(t *testing.T) {
	if got := nextBackoff(8*time.Second, 10*time.Second); got != 10*time.Second {
		t.Fatalf("nextBackoff() = %s, want 10s", got)
	}
}
