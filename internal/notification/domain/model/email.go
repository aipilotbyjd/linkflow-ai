// Package model defines notification domain models
package model

import (
	"time"

	"github.com/google/uuid"
)

// EmailType represents types of emails
type EmailType string

const (
	EmailTypePasswordReset       EmailType = "password_reset"
	EmailTypeEmailVerification   EmailType = "email_verification"
	EmailTypeWorkspaceInvitation EmailType = "workspace_invitation"
	EmailTypePaymentSuccess      EmailType = "payment_success"
	EmailTypePaymentFailed       EmailType = "payment_failed"
	EmailTypeTrialEnding         EmailType = "trial_ending"
	EmailTypeExecutionAlert      EmailType = "execution_alert"
	EmailTypeUsageLimitWarning   EmailType = "usage_limit_warning"
	EmailTypeWelcome             EmailType = "welcome"
	EmailTypeTeamUpdate          EmailType = "team_update"
)

// EmailStatus represents email sending status
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusFailed    EmailStatus = "failed"
	EmailStatusDelivered EmailStatus = "delivered"
	EmailStatusBounced   EmailStatus = "bounced"
)

// Email represents an email to be sent
type Email struct {
	ID          string
	Type        EmailType
	Status      EmailStatus
	To          string
	ToName      string
	From        string
	FromName    string
	ReplyTo     string
	Subject     string
	TextContent string
	HTMLContent string
	TemplateID  string
	Variables   map[string]interface{}
	Metadata    map[string]string
	SentAt      *time.Time
	Error       string
	Retries     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewEmail creates a new email
func NewEmail(emailType EmailType, to, toName, subject string) *Email {
	now := time.Now()
	return &Email{
		ID:        uuid.New().String(),
		Type:      emailType,
		Status:    EmailStatusPending,
		To:        to,
		ToName:    toName,
		Subject:   subject,
		Variables: make(map[string]interface{}),
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MarkSent marks email as sent
func (e *Email) MarkSent() {
	now := time.Now()
	e.Status = EmailStatusSent
	e.SentAt = &now
	e.UpdatedAt = now
}

// MarkFailed marks email as failed
func (e *Email) MarkFailed(err string) {
	e.Status = EmailStatusFailed
	e.Error = err
	e.Retries++
	e.UpdatedAt = time.Now()
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID           string
	Name         string
	Type         EmailType
	Subject      string
	TextTemplate string
	HTMLTemplate string
	Variables    []string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DefaultTemplates returns default email templates
func DefaultTemplates() map[EmailType]*EmailTemplate {
	return map[EmailType]*EmailTemplate{
		EmailTypePasswordReset: {
			Type:    EmailTypePasswordReset,
			Name:    "Password Reset",
			Subject: "Reset your password",
			HTMLTemplate: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Reset Your Password</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #1a1a2e;">Reset Your Password</h1>
    <p>Hi {{.Name}},</p>
    <p>You requested to reset your password. Click the button below to set a new password:</p>
    <p style="text-align: center; margin: 30px 0;">
        <a href="{{.ResetURL}}" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">Reset Password</a>
    </p>
    <p>This link will expire in 1 hour.</p>
    <p>If you didn't request this, you can safely ignore this email.</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
    <p style="color: #6b7280; font-size: 12px;">LinkFlow AI</p>
</body>
</html>`,
			TextTemplate: `Reset Your Password

Hi {{.Name}},

You requested to reset your password. Visit this link to set a new password:
{{.ResetURL}}

This link will expire in 1 hour.

If you didn't request this, you can safely ignore this email.

LinkFlow AI`,
			Variables: []string{"Name", "ResetURL"},
		},
		EmailTypeEmailVerification: {
			Type:    EmailTypeEmailVerification,
			Name:    "Email Verification",
			Subject: "Verify your email address",
			HTMLTemplate: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Verify Your Email</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #1a1a2e;">Verify Your Email</h1>
    <p>Hi {{.Name}},</p>
    <p>Thanks for signing up! Please verify your email address by clicking the button below:</p>
    <p style="text-align: center; margin: 30px 0;">
        <a href="{{.VerifyURL}}" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">Verify Email</a>
    </p>
    <p>This link will expire in 24 hours.</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
    <p style="color: #6b7280; font-size: 12px;">LinkFlow AI</p>
</body>
</html>`,
			TextTemplate: `Verify Your Email

Hi {{.Name}},

Thanks for signing up! Please verify your email address by visiting:
{{.VerifyURL}}

This link will expire in 24 hours.

LinkFlow AI`,
			Variables: []string{"Name", "VerifyURL"},
		},
		EmailTypeWorkspaceInvitation: {
			Type:    EmailTypeWorkspaceInvitation,
			Name:    "Workspace Invitation",
			Subject: "You've been invited to join {{.WorkspaceName}}",
			HTMLTemplate: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Workspace Invitation</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #1a1a2e;">You're Invited!</h1>
    <p>Hi there,</p>
    <p><strong>{{.InviterName}}</strong> has invited you to join <strong>{{.WorkspaceName}}</strong> on LinkFlow AI.</p>
    <p>You've been invited as a <strong>{{.Role}}</strong>.</p>
    <p style="text-align: center; margin: 30px 0;">
        <a href="{{.InviteURL}}" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">Accept Invitation</a>
    </p>
    <p>This invitation will expire in 7 days.</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
    <p style="color: #6b7280; font-size: 12px;">LinkFlow AI</p>
</body>
</html>`,
			TextTemplate: `You're Invited!

Hi there,

{{.InviterName}} has invited you to join {{.WorkspaceName}} on LinkFlow AI.

You've been invited as a {{.Role}}.

Accept the invitation: {{.InviteURL}}

This invitation will expire in 7 days.

LinkFlow AI`,
			Variables: []string{"InviterName", "WorkspaceName", "Role", "InviteURL"},
		},
		EmailTypeWelcome: {
			Type:    EmailTypeWelcome,
			Name:    "Welcome",
			Subject: "Welcome to LinkFlow AI!",
			HTMLTemplate: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Welcome</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #1a1a2e;">Welcome to LinkFlow AI!</h1>
    <p>Hi {{.Name}},</p>
    <p>Thanks for joining LinkFlow AI! We're excited to have you.</p>
    <p>Here's what you can do to get started:</p>
    <ul>
        <li>Create your first workflow</li>
        <li>Connect your apps and services</li>
        <li>Invite your team members</li>
    </ul>
    <p style="text-align: center; margin: 30px 0;">
        <a href="{{.DashboardURL}}" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">Go to Dashboard</a>
    </p>
    <p>Need help? Check out our <a href="{{.DocsURL}}">documentation</a> or reach out to support.</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
    <p style="color: #6b7280; font-size: 12px;">LinkFlow AI</p>
</body>
</html>`,
			TextTemplate: `Welcome to LinkFlow AI!

Hi {{.Name}},

Thanks for joining LinkFlow AI! We're excited to have you.

Here's what you can do to get started:
- Create your first workflow
- Connect your apps and services
- Invite your team members

Go to your dashboard: {{.DashboardURL}}

Need help? Check out our documentation: {{.DocsURL}}

LinkFlow AI`,
			Variables: []string{"Name", "DashboardURL", "DocsURL"},
		},
	}
}
