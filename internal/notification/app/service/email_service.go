// Package service provides email business logic
package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
)

// EmailProvider defines email sending interface
type EmailProvider interface {
	Send(ctx context.Context, email *model.Email) error
}

// EmailRepository defines email persistence
type EmailRepository interface {
	Create(ctx context.Context, email *model.Email) error
	FindByID(ctx context.Context, id string) (*model.Email, error)
	Update(ctx context.Context, email *model.Email) error
	ListPending(ctx context.Context, limit int) ([]*model.Email, error)
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	FromAddress    string
	FromName       string
	ReplyTo        string
	BaseURL        string
	AppName        string
	MaxRetries     int
	RetryInterval  time.Duration
}

// EmailService handles email operations
type EmailService struct {
	provider   EmailProvider
	repository EmailRepository
	config     EmailConfig
	templates  map[model.EmailType]*model.EmailTemplate
}

// NewEmailService creates a new email service
func NewEmailService(provider EmailProvider, repository EmailRepository, config EmailConfig) *EmailService {
	return &EmailService{
		provider:   provider,
		repository: repository,
		config:     config,
		templates:  model.DefaultTemplates(),
	}
}

// SendPasswordReset sends password reset email
func (s *EmailService) SendPasswordReset(ctx context.Context, to, name, token string) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", s.config.BaseURL, token)
	
	return s.sendTemplateEmail(ctx, model.EmailTypePasswordReset, to, name, map[string]interface{}{
		"Name":     name,
		"ResetURL": resetURL,
	})
}

// SendEmailVerification sends email verification email
func (s *EmailService) SendEmailVerification(ctx context.Context, to, name, token string) error {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", s.config.BaseURL, token)
	
	return s.sendTemplateEmail(ctx, model.EmailTypeEmailVerification, to, name, map[string]interface{}{
		"Name":      name,
		"VerifyURL": verifyURL,
	})
}

// SendWorkspaceInvitation sends workspace invitation email
func (s *EmailService) SendWorkspaceInvitation(ctx context.Context, to, workspaceName, inviterName, role, token string) error {
	inviteURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.config.BaseURL, token)
	
	return s.sendTemplateEmail(ctx, model.EmailTypeWorkspaceInvitation, to, "", map[string]interface{}{
		"WorkspaceName": workspaceName,
		"InviterName":   inviterName,
		"Role":          role,
		"InviteURL":     inviteURL,
	})
}

// SendWelcome sends welcome email
func (s *EmailService) SendWelcome(ctx context.Context, to, name string) error {
	dashboardURL := fmt.Sprintf("%s/dashboard", s.config.BaseURL)
	docsURL := fmt.Sprintf("%s/docs", s.config.BaseURL)
	
	return s.sendTemplateEmail(ctx, model.EmailTypeWelcome, to, name, map[string]interface{}{
		"Name":         name,
		"DashboardURL": dashboardURL,
		"DocsURL":      docsURL,
	})
}

// SendPaymentSuccess sends payment success email
func (s *EmailService) SendPaymentSuccess(ctx context.Context, to, name, planName string, amount int64) error {
	email := model.NewEmail(model.EmailTypePaymentSuccess, to, name, "Payment successful")
	email.From = s.config.FromAddress
	email.FromName = s.config.FromName
	email.Variables = map[string]interface{}{
		"Name":     name,
		"PlanName": planName,
		"Amount":   fmt.Sprintf("$%.2f", float64(amount)/100),
	}
	
	// Render basic email
	email.HTMLContent = fmt.Sprintf(`
		<h1>Payment Successful</h1>
		<p>Hi %s,</p>
		<p>Your payment of %s for the %s plan has been processed successfully.</p>
		<p>Thank you for your subscription!</p>
	`, name, email.Variables["Amount"], planName)
	
	email.TextContent = fmt.Sprintf(`Payment Successful

Hi %s,

Your payment of %s for the %s plan has been processed successfully.

Thank you for your subscription!
`, name, email.Variables["Amount"], planName)

	return s.queueAndSend(ctx, email)
}

// SendPaymentFailed sends payment failed email
func (s *EmailService) SendPaymentFailed(ctx context.Context, to, name, reason string) error {
	email := model.NewEmail(model.EmailTypePaymentFailed, to, name, "Payment failed - action required")
	email.From = s.config.FromAddress
	email.FromName = s.config.FromName
	
	updateURL := fmt.Sprintf("%s/settings/billing", s.config.BaseURL)
	
	email.HTMLContent = fmt.Sprintf(`
		<h1>Payment Failed</h1>
		<p>Hi %s,</p>
		<p>We were unable to process your payment. Reason: %s</p>
		<p>Please update your payment method to avoid service interruption:</p>
		<p><a href="%s">Update Payment Method</a></p>
	`, name, reason, updateURL)
	
	email.TextContent = fmt.Sprintf(`Payment Failed

Hi %s,

We were unable to process your payment. Reason: %s

Please update your payment method to avoid service interruption:
%s
`, name, reason, updateURL)

	return s.queueAndSend(ctx, email)
}

// SendTrialEnding sends trial ending reminder
func (s *EmailService) SendTrialEnding(ctx context.Context, to, name string, daysLeft int) error {
	email := model.NewEmail(model.EmailTypeTrialEnding, to, name, fmt.Sprintf("Your trial ends in %d days", daysLeft))
	email.From = s.config.FromAddress
	email.FromName = s.config.FromName
	
	upgradeURL := fmt.Sprintf("%s/settings/billing/upgrade", s.config.BaseURL)
	
	email.HTMLContent = fmt.Sprintf(`
		<h1>Your Trial Is Ending Soon</h1>
		<p>Hi %s,</p>
		<p>Your free trial will end in %d days. To continue using all features, please upgrade to a paid plan.</p>
		<p><a href="%s">Upgrade Now</a></p>
	`, name, daysLeft, upgradeURL)
	
	email.TextContent = fmt.Sprintf(`Your Trial Is Ending Soon

Hi %s,

Your free trial will end in %d days. To continue using all features, please upgrade to a paid plan.

Upgrade now: %s
`, name, daysLeft, upgradeURL)

	return s.queueAndSend(ctx, email)
}

// SendExecutionAlert sends workflow execution alert
func (s *EmailService) SendExecutionAlert(ctx context.Context, to, name, workflowName, status, errorMsg string) error {
	subject := fmt.Sprintf("Workflow '%s' %s", workflowName, status)
	email := model.NewEmail(model.EmailTypeExecutionAlert, to, name, subject)
	email.From = s.config.FromAddress
	email.FromName = s.config.FromName
	
	email.HTMLContent = fmt.Sprintf(`
		<h1>Workflow Execution Alert</h1>
		<p>Hi %s,</p>
		<p>Workflow <strong>%s</strong> has %s.</p>
		%s
	`, name, workflowName, status, func() string {
		if errorMsg != "" {
			return fmt.Sprintf("<p>Error: %s</p>", errorMsg)
		}
		return ""
	}())
	
	return s.queueAndSend(ctx, email)
}

// SendUsageLimitWarning sends usage limit warning
func (s *EmailService) SendUsageLimitWarning(ctx context.Context, to, name, limitType string, currentUsage, limit int) error {
	percentage := float64(currentUsage) / float64(limit) * 100
	subject := fmt.Sprintf("Usage warning: %.0f%% of %s limit reached", percentage, limitType)
	
	email := model.NewEmail(model.EmailTypeUsageLimitWarning, to, name, subject)
	email.From = s.config.FromAddress
	email.FromName = s.config.FromName
	
	upgradeURL := fmt.Sprintf("%s/settings/billing/upgrade", s.config.BaseURL)
	
	email.HTMLContent = fmt.Sprintf(`
		<h1>Usage Limit Warning</h1>
		<p>Hi %s,</p>
		<p>You've used %d of your %d %s limit (%.0f%%).</p>
		<p>Consider upgrading your plan to avoid interruptions:</p>
		<p><a href="%s">Upgrade Plan</a></p>
	`, name, currentUsage, limit, limitType, percentage, upgradeURL)
	
	return s.queueAndSend(ctx, email)
}

// sendTemplateEmail renders and sends a templated email
func (s *EmailService) sendTemplateEmail(ctx context.Context, emailType model.EmailType, to, toName string, vars map[string]interface{}) error {
	tmpl, ok := s.templates[emailType]
	if !ok {
		return fmt.Errorf("template not found for type: %s", emailType)
	}
	
	// Render subject
	subject, err := s.renderTemplate(tmpl.Subject, vars)
	if err != nil {
		return fmt.Errorf("failed to render subject: %w", err)
	}
	
	email := model.NewEmail(emailType, to, toName, subject)
	email.From = s.config.FromAddress
	email.FromName = s.config.FromName
	email.ReplyTo = s.config.ReplyTo
	email.Variables = vars
	
	// Render HTML content
	htmlContent, err := s.renderTemplate(tmpl.HTMLTemplate, vars)
	if err != nil {
		return fmt.Errorf("failed to render HTML template: %w", err)
	}
	email.HTMLContent = htmlContent
	
	// Render text content
	textContent, err := s.renderTemplate(tmpl.TextTemplate, vars)
	if err != nil {
		return fmt.Errorf("failed to render text template: %w", err)
	}
	email.TextContent = textContent
	
	return s.queueAndSend(ctx, email)
}

func (s *EmailService) renderTemplate(tmplStr string, vars map[string]interface{}) (string, error) {
	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

func (s *EmailService) queueAndSend(ctx context.Context, email *model.Email) error {
	// Save to repository first
	if s.repository != nil {
		if err := s.repository.Create(ctx, email); err != nil {
			return fmt.Errorf("failed to queue email: %w", err)
		}
	}
	
	// Try to send immediately
	if err := s.provider.Send(ctx, email); err != nil {
		email.MarkFailed(err.Error())
		if s.repository != nil {
			s.repository.Update(ctx, email)
		}
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	email.MarkSent()
	if s.repository != nil {
		s.repository.Update(ctx, email)
	}
	
	return nil
}

// ProcessPendingEmails processes pending emails in queue
func (s *EmailService) ProcessPendingEmails(ctx context.Context) error {
	if s.repository == nil {
		return nil
	}
	
	pending, err := s.repository.ListPending(ctx, 100)
	if err != nil {
		return err
	}
	
	for _, email := range pending {
		if email.Retries >= s.config.MaxRetries {
			continue
		}
		
		if err := s.provider.Send(ctx, email); err != nil {
			email.MarkFailed(err.Error())
		} else {
			email.MarkSent()
		}
		
		s.repository.Update(ctx, email)
	}
	
	return nil
}
