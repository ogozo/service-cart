package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
)

type TraceCarrier map[string]interface{}

func (c TraceCarrier) Get(key string) string {
	if val, ok := c[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (c TraceCarrier) Set(key, val string) {
	c[key] = val
}

func (c TraceCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tracer  trace.Tracer
}

func NewConsumer(amqpURL string) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}
	tracer := otel.Tracer("service-cart.broker")
	return &Consumer{conn: conn, channel: channel, tracer: tracer}, nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

type OrderConfirmedEvent struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func (c *Consumer) StartOrderConfirmedConsumer(handler func(ctx context.Context, event OrderConfirmedEvent)) error {
	exchangeName := "order_confirmed_exchange"
	err := c.channel.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil)
	if err != nil {
		return err
	}

	q, err := c.channel.QueueDeclare("cart_service_order_confirmed_queue", true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = c.channel.QueueBind(q.Name, "", exchangeName, false, nil)
	if err != nil {
		return err
	}

	msgs, err := c.channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			carrier := make(TraceCarrier)
			for k, v := range d.Headers {
				carrier[k] = v
			}
			parentCtx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

			spanCtx, span := c.tracer.Start(parentCtx, exchangeName+" receive", trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					semconv.MessagingSystemRabbitmq,
					semconv.MessagingDestinationName(exchangeName),
				),
			)

			log.Printf("📩 Received OrderConfirmed event: %s", d.Body)
			var event OrderConfirmedEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("Error unmarshalling event: %v", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to unmarshal message")
				span.End()
				continue
			}

			handler(spanCtx, event)
			span.End()
		}
	}()
	log.Println("👂 Listening for OrderConfirmed events...")
	return nil
}
