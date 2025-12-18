// Package smtp provides SMTP email sending implementation
package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	UseTLS     bool
	SkipVerify bool
}

// SMTPProvider implements email sending via SMTP
type SMTPProvider struct {
	config SMTPConfig
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(config SMTPConfig) *SMTPProvider {
	return &SMTPProvider{config: config}
}

// Send sends an email via SMTP
func (p *SMTPProvider) Send(ctx context.Context, email *model.Email) error {
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
	
	// Build message
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", email.FromName, email.From)
	headers["To"] = email.To
	headers["Subject"] = email.Subject
	headers["MIME-Version"] = "1.0"
	
	// Build multipart message if we have both HTML and text
	var message string
	if email.HTMLContent != "" && email.TextContent != "" {
		boundary := "----=_Part_0_1234567890"
		headers["Content-Type"] = fmt.Sprintf("multipart/alternative; boundary=\"%s\"", boundary)
		
		message = buildHeaders(headers)
		message += fmt.Sprintf("\r\n--%s\r\n", boundary)
		message += "Content-Type: text/plain; charset=UTF-8\r\n\r\n"
		message += email.TextContent
		message += fmt.Sprintf("\r\n--%s\r\n", boundary)
		message += "Content-Type: text/html; charset=UTF-8\r\n\r\n"
		message += email.HTMLContent
		message += fmt.Sprintf("\r\n--%s--", boundary)
	} else if email.HTMLContent != "" {
		headers["Content-Type"] = "text/html; charset=UTF-8"
		message = buildHeaders(headers) + "\r\n" + email.HTMLContent
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
		message = buildHeaders(headers) + "\r\n" + email.TextContent
	}
	
	// Auth
	var auth smtp.Auth
	if p.config.Username != "" {
		auth = smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
	}
	
	// Connect and send
	if p.config.UseTLS {
		return p.sendTLS(addr, auth, email.From, email.To, []byte(message))
	}
	
	return smtp.SendMail(addr, auth, email.From, []string{email.To}, []byte(message))
}

func (p *SMTPProvider) sendTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: p.config.SkipVerify,
		ServerName:         p.config.Host,
	}
	
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Quit()
	
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL command failed: %w", err)
	}
	
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT command failed: %w", err)
	}
	
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}
	
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("write data failed: %w", err)
	}
	
	return w.Close()
}

func buildHeaders(headers map[string]string) string {
	result := ""
	for k, v := range headers {
		result += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	return result
}
