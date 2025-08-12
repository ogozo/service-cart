package broker

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewConsumer(amqpURL string) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil { return nil, fmt.Errorf("failed to connect: %w", err) }
	channel, err := conn.Channel()
	if err != nil { return nil, fmt.Errorf("failed to open channel: %w", err) }
	return &Consumer{conn: conn, channel: channel}, nil
}

type OrderConfirmedEvent struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func (c *Consumer) StartOrderConfirmedConsumer(handler func(event OrderConfirmedEvent)) error {
	err := c.channel.ExchangeDeclare("order_confirmed_exchange", "fanout", true, false, false, false, nil)
	if err != nil { return err }

	q, err := c.channel.QueueDeclare("cart_service_order_confirmed_queue", true, false, false, false, nil)
	if err != nil { return err }

	err = c.channel.QueueBind(q.Name, "", "order_confirmed_exchange", false, nil)
	if err != nil { return err }

	msgs, err := c.channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil { return err }

	go func() {
		for d := range msgs {
			log.Printf("📩 Received OrderConfirmed event: %s", d.Body)
			var event OrderConfirmedEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("Error unmarshalling event: %v", err)
				continue
			}
			handler(event)
		}
	}()
    log.Println("👂 Listening for OrderConfirmed events...")
	return nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		} else {
			log.Println("RabbitMQ channel closed successfully.")
		}
	}
	
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		} else {
			log.Println("RabbitMQ connection closed successfully.")
		}
	}
}
