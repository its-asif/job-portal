package queue

import (
	"context"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client wraps the AMQP connection and channel.
type Client struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// NewRabbitMQClient connects to RabbitMQ using RABBITMQ_URL.
// If RABBITMQ_URL is empty, it falls back to amqp://guest:guest@localhost:5672/.
func NewRabbitMQClient() (*Client, error) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Client{Conn: conn, Channel: ch}, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.Channel != nil {
		_ = c.Channel.Close()
	}
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// Publish sends a durable message to a durable queue.
func (c *Client) Publish(queueName, message string) error {
	if c == nil || c.Channel == nil {
		return amqp.ErrClosed
	}

	if _, err := c.Channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Channel.PublishWithContext(
		ctx,
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         []byte(message),
		},
	)
}
