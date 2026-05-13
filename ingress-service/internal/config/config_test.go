package config

import (
	"strings"
	"testing"
)

func TestValidateAllowsDevelopmentDefaultsOutsideProd(t *testing.T) {
	cfg := Config{
		AppEnv:           "dev",
		PostgresDSN:      DevPostgresDSN,
		RabbitMQURL:      DevRabbitMQURL,
		CodeupToken:      DevCodeupToken,
		ProjexSecret:     DevProjexSecret,
		FlowToken:        DevFlowToken,
		RabbitMQExchange: "",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsDevelopmentDefaultsInProd(t *testing.T) {
	cfg := Config{
		AppEnv:           "prod",
		PostgresDSN:      DevPostgresDSN,
		RabbitMQURL:      DevRabbitMQURL,
		CodeupToken:      DevCodeupToken,
		ProjexSecret:     DevProjexSecret,
		FlowToken:        DevFlowToken,
		RabbitMQExchange: "yunxiao.events",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error is nil, want failure")
	}
	message := err.Error()
	for _, expected := range []string{
		"POSTGRES_DSN must not use development default",
		"RABBITMQ_URL must not use development default",
		"CODEUP_WEBHOOK_TOKEN must not use development default",
		"PROJEX_WEBHOOK_SECRET must not use development default",
		"FLOW_WEBHOOK_TOKEN must not use development default",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error = %q, want %q", message, expected)
		}
	}
}

func TestValidateAllowsExplicitProductionConfig(t *testing.T) {
	cfg := Config{
		AppEnv:           "prod",
		PostgresDSN:      "postgres://prod:secret@postgres:5432/prod?sslmode=disable",
		RabbitMQURL:      "amqp://prod:secret@rabbitmq:5672/",
		CodeupToken:      "codeup-prod-token",
		ProjexSecret:     "projex-prod-secret",
		FlowToken:        "flow-prod-token",
		RabbitMQExchange: "yunxiao.events",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
