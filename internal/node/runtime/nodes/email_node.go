// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// EmailNode implements email sending
type EmailNode struct{}

// NewEmailNode creates a new Email node
func NewEmailNode() *EmailNode {
	return &EmailNode{}
}

// GetType returns the node type
func (n *EmailNode) GetType() string {
	return "email"
}

// GetMetadata returns node metadata
func (n *EmailNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "email",
		Name:        "Send Email",
		Description: "Send emails via SMTP",
		Category:    "communication",
		Icon:        "mail",
		Color:       "#EA4335",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Send result"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "fromEmail", Type: "string", Required: true, Description: "From email address"},
			{Name: "fromName", Type: "string", Description: "From name"},
			{Name: "toEmail", Type: "string", Required: true, Description: "To email address(es), comma-separated"},
			{Name: "ccEmail", Type: "string", Description: "CC email address(es)"},
			{Name: "bccEmail", Type: "string", Description: "BCC email address(es)"},
			{Name: "replyTo", Type: "string", Description: "Reply-to address"},
			{Name: "subject", Type: "string", Required: true, Description: "Email subject"},
			{Name: "textContent", Type: "string", Description: "Plain text content"},
			{Name: "htmlContent", Type: "code", Description: "HTML content"},
			{Name: "smtpHost", Type: "string", Default: "smtp.gmail.com", Description: "SMTP host"},
			{Name: "smtpPort", Type: "number", Default: 587, Description: "SMTP port"},
			{Name: "useTLS", Type: "boolean", Default: true, Description: "Use TLS"},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *EmailNode) Validate(config map[string]interface{}) error {
	if getStringConfig(config, "toEmail", "") == "" {
		return fmt.Errorf("toEmail is required")
	}
	if getStringConfig(config, "subject", "") == "" {
		return fmt.Errorf("subject is required")
	}
	if getStringConfig(config, "textContent", "") == "" && getStringConfig(config, "htmlContent", "") == "" {
		return fmt.Errorf("either textContent or htmlContent is required")
	}
	return nil
}

// Execute executes the Email node
func (n *EmailNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	// Get SMTP credentials
	username := ""
	password := ""
	if input.Credentials != nil {
		username, _ = input.Credentials["username"].(string)
		password, _ = input.Credentials["password"].(string)
	}
	
	// Get config
	fromEmail := getStringConfig(input.NodeConfig, "fromEmail", "")
	fromName := getStringConfig(input.NodeConfig, "fromName", "")
	toEmail := getStringConfig(input.NodeConfig, "toEmail", "")
	ccEmail := getStringConfig(input.NodeConfig, "ccEmail", "")
	bccEmail := getStringConfig(input.NodeConfig, "bccEmail", "")
	replyTo := getStringConfig(input.NodeConfig, "replyTo", "")
	subject := getStringConfig(input.NodeConfig, "subject", "")
	textContent := getStringConfig(input.NodeConfig, "textContent", "")
	htmlContent := getStringConfig(input.NodeConfig, "htmlContent", "")
	smtpHost := getStringConfig(input.NodeConfig, "smtpHost", "smtp.gmail.com")
	smtpPort := getIntConfig(input.NodeConfig, "smtpPort", 587)
	useTLS := getBoolConfig(input.NodeConfig, "useTLS", true)
	
	// Build recipient list
	recipients := parseEmailList(toEmail)
	recipients = append(recipients, parseEmailList(ccEmail)...)
	recipients = append(recipients, parseEmailList(bccEmail)...)
	
	// Build message
	var msg strings.Builder
	
	// From header
	if fromName != "" {
		msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail))
	} else {
		msg.WriteString(fmt.Sprintf("From: %s\r\n", fromEmail))
	}
	
	// To header
	msg.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	
	// CC header
	if ccEmail != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", ccEmail))
	}
	
	// Reply-To header
	if replyTo != "" {
		msg.WriteString(fmt.Sprintf("Reply-To: %s\r\n", replyTo))
	}
	
	// Subject
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	
	// MIME headers for multipart
	if htmlContent != "" && textContent != "" {
		boundary := "----=_Part_0_" + fmt.Sprintf("%d", time.Now().UnixNano())
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		msg.WriteString("\r\n")
		
		// Plain text part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(textContent)
		msg.WriteString("\r\n")
		
		// HTML part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		msg.WriteString(htmlContent)
		msg.WriteString("\r\n")
		
		msg.WriteString(fmt.Sprintf("--%s--", boundary))
	} else if htmlContent != "" {
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		msg.WriteString(htmlContent)
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(textContent)
	}
	
	// Send email
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	
	var err error
	if useTLS {
		err = n.sendTLS(addr, smtpHost, username, password, fromEmail, recipients, msg.String())
	} else {
		auth := smtp.PlainAuth("", username, password, smtpHost)
		err = smtp.SendMail(addr, auth, fromEmail, recipients, []byte(msg.String()))
	}
	
	if err != nil {
		output.Error = fmt.Errorf("failed to send email: %w", err)
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("Email send failed: %v", err),
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
		return output, nil
	}
	
	output.Data = map[string]interface{}{
		"success":    true,
		"to":         toEmail,
		"subject":    subject,
		"recipients": len(recipients),
		"sentAt":     time.Now().Format(time.RFC3339),
	}
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Email sent to %d recipients", len(recipients)),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

func (n *EmailNode) sendTLS(addr, host, username, password, from string, to []string, msg string) error {
	tlsConfig := &tls.Config{
		ServerName: host,
	}
	
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// Try STARTTLS
		return n.sendSTARTTLS(addr, host, username, password, from, to, msg)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()
	
	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	
	if err := client.Mail(from); err != nil {
		return err
	}
	
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	
	w, err := client.Data()
	if err != nil {
		return err
	}
	
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	
	return w.Close()
}

func (n *EmailNode) sendSTARTTLS(addr, host, username, password, from string, to []string, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Quit()
	
	tlsConfig := &tls.Config{
		ServerName: host,
	}
	
	if err := client.StartTLS(tlsConfig); err != nil {
		return err
	}
	
	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	
	if err := client.Mail(from); err != nil {
		return err
	}
	
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	
	w, err := client.Data()
	if err != nil {
		return err
	}
	
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	
	return w.Close()
}

func parseEmailList(emails string) []string {
	if emails == "" {
		return []string{}
	}
	
	parts := strings.Split(emails, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		email := strings.TrimSpace(p)
		if email != "" {
			result = append(result, email)
		}
	}
	return result
}

func init() {
	runtime.Register(NewEmailNode())
}
