package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/adapters/http/middleware"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/app/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")

	// Public routes
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/logout", h.Logout)
	auth.POST("/password/forgot", h.ForgotPassword)
	auth.POST("/password/reset", h.ResetPassword)
	auth.POST("/email/verify", h.VerifyEmail)

	// Protected routes
	protected := auth.Group("")
	protected.Use(middleware.Auth(h.authService))
	{
		protected.POST("/password/change", h.ChangePassword)
		protected.POST("/email/resend", h.ResendVerification)
		protected.GET("/api-keys", h.ListAPIKeys)
		protected.POST("/api-keys", h.CreateAPIKey)
		protected.DELETE("/api-keys/:id", h.RevokeAPIKey)
		protected.POST("/mfa/setup", h.MFASetup)
		protected.POST("/mfa/verify", h.MFAVerify)
		protected.POST("/mfa/disable", h.MFADisable)
	}

	// OAuth routes
	auth.GET("/oauth/google", h.OAuthGoogle)
	auth.GET("/oauth/github", h.OAuthGitHub)
	auth.GET("/oauth/callback", h.OAuthCallback)
}
