// Package sendgrid provides SendGrid email sending implementation
package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
)

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey string
}

// SendGridProvider implements email sending via SendGrid
type SendGridProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider(config SendGridConfig) *SendGridProvider {
	return &SendGridProvider{
		apiKey:     config.APIKey,
		httpClient: &http.Client{},
	}
}

// sendGridRequest represents SendGrid API request
type sendGridRequest struct {
	Personalizations []personalization `json:"personalizations"`
	From             emailAddress      `json:"from"`
	ReplyTo          *emailAddress     `json:"reply_to,omitempty"`
	Subject          string            `json:"subject"`
	Content          []content         `json:"content"`
}

type personalization struct {
	To []emailAddress `json:"to"`
}

type emailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type content struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Send sends an email via SendGrid
func (p *SendGridProvider) Send(ctx context.Context, email *model.Email) error {
	req := sendGridRequest{
		Personalizations: []personalization{
			{
				To: []emailAddress{
					{Email: email.To, Name: email.ToName},
				},
			},
		},
		From:    emailAddress{Email: email.From, Name: email.FromName},
		Subject: email.Subject,
		Content: []content{},
	}
	
	if email.ReplyTo != "" {
		req.ReplyTo = &emailAddress{Email: email.ReplyTo}
	}
	
	if email.TextContent != "" {
		req.Content = append(req.Content, content{Type: "text/plain", Value: email.TextContent})
	}
	if email.HTMLContent != "" {
		req.Content = append(req.Content, content{Type: "text/html", Value: email.HTMLContent})
	}
	
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d", resp.StatusCode)
	}
	
	return nil
}
