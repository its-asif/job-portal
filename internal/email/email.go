package email

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	gomail "gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	host := strings.TrimSpace(os.Getenv("MAIL_HOST"))
	portStr := strings.TrimSpace(os.Getenv("MAIL_PORT"))
	username := strings.TrimSpace(os.Getenv("MAIL_USERNAME"))
	password := os.Getenv("MAIL_PASSWORD")
	from := strings.TrimSpace(os.Getenv("MAIL_FROM"))

	if host == "" || portStr == "" || username == "" || password == "" || from == "" {
		return fmt.Errorf("mail configuration is incomplete")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid MAIL_PORT: %w", err)
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	dialer := gomail.NewDialer(host, port, username, password)
	return dialer.DialAndSend(msg)
}
