package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/its-asif/job-portal/internal/email"
	"github.com/its-asif/job-portal/internal/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type notificationEvent struct {
	Type           string `json:"type"`
	JobID          int    `json:"job_id"`
	ApplicantID    int    `json:"applicant_id"`
	ApplicantEmail string `json:"applicant_email"`
}

func main() {
	loadEnvFile(".env")

	rabbitClient, err := queue.NewRabbitMQClient()
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer func() {
		if closeErr := rabbitClient.Close(); closeErr != nil {
			log.Printf("failed to close rabbitmq connection: %v", closeErr)
		}
	}()

	q, err := rabbitClient.Channel.QueueDeclare(
		"notifications",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare queue: %v", err)
	}

	failedQ, err := rabbitClient.Channel.QueueDeclare(
		"failed_notifications",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare failed queue: %v", err)
	}

	msgs, err := rabbitClient.Channel.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	log.Printf("worker started, consuming queue: %s", q.Name)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigCh:
			log.Printf("received signal %s, stopping worker", sig)
			return
		case d, ok := <-msgs:
			if !ok {
				log.Printf("message channel closed, stopping worker")
				return
			}

			handleDelivery(rabbitClient.Channel, failedQ.Name, d)
		}
	}
}

func handleDelivery(ch *amqp.Channel, failedQueue string, d amqp.Delivery) {
	var event notificationEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("invalid notification payload: %s", string(d.Body))
		if publishErr := publishWithHeaders(ch, failedQueue, d.Body, amqp.Table{
			"x-last-error":     "invalid payload",
			"x-retry-count":    int32(retryCountFromHeaders(d.Headers)),
			"x-original-queue": "notifications",
		}); publishErr != nil {
			log.Printf("failed to move invalid payload to dead letter queue: %v", publishErr)
			_ = d.Nack(false, true)
			return
		}
		_ = d.Ack(false)
		return
	}

	subject := "You have a new application"
	body := fmt.Sprintf("New application received for job ID %d from applicant ID %d", event.JobID, event.ApplicantID)

	if err := email.SendEmail(event.ApplicantEmail, subject, body); err == nil {
		log.Printf("email sent to: %s", event.ApplicantEmail)
		_ = d.Ack(false)
		return
	} else {
		retryCount := retryCountFromHeaders(d.Headers)
		if retryCount < 3 {
			headers := amqp.Table{
				"x-retry-count": int32(retryCount + 1),
				"x-last-error":  err.Error(),
			}
			if publishErr := publishWithHeaders(ch, "notifications", d.Body, headers); publishErr != nil {
				log.Printf("failed to republish retry message: %v", publishErr)
				_ = d.Nack(false, true)
				return
			}
			log.Printf("email send failed, retry %d queued for %s", retryCount+1, event.ApplicantEmail)
			_ = d.Ack(false)
			return
		}

		headers := amqp.Table{
			"x-retry-count": int32(retryCount),
			"x-last-error":  err.Error(),
		}
		if publishErr := publishWithHeaders(ch, failedQueue, d.Body, headers); publishErr != nil {
			log.Printf("failed to publish to dead letter queue: %v", publishErr)
			_ = d.Nack(false, true)
			return
		}
		log.Printf("moved message to failed_notifications after %d retries", retryCount)
		_ = d.Ack(false)
	}
}

func publishWithHeaders(ch *amqp.Channel, queueName string, body []byte, headers amqp.Table) error {
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ch.PublishWithContext(
		ctx,
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Body:         body,
		},
	)
}

func retryCountFromHeaders(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	raw, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}

	switch v := raw.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return v
	case int32:
		if v < 0 {
			return 0
		}
		return int(v)
	case int64:
		if v < 0 {
			return 0
		}
		return int(v)
	case float64:
		if v < 0 {
			return 0
		}
		return int(v)
	default:
		return 0
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == "" {
			continue
		}

		if setErr := os.Setenv(key, value); setErr != nil {
			log.Printf("failed to set env var %s: %v", key, setErr)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("failed to read env file %s: %v", path, err)
	}
}
