package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/domain/model"
)

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authService.Register(c.Request.Context(), service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrEmailExists {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"accessToken":  result.Tokens.AccessToken,
		"refreshToken": result.Tokens.RefreshToken,
		"expiresAt":    result.Tokens.ExpiresAt,
		"tokenType":    result.Tokens.TokenType,
		"user": gin.H{
			"id":            result.User.ID,
			"email":         result.User.Email,
			"firstName":     result.User.FirstName,
			"lastName":      result.User.LastName,
			"emailVerified": result.User.EmailVerified,
		},
	})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	MFACode  string `json:"mfaCode,omitempty"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		MFACode:   req.MFACode,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		status := http.StatusUnauthorized
		if err == model.ErrAccountLocked {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	if result.RequiresMFA {
		c.JSON(http.StatusOK, gin.H{"requiresMFA": true, "mfaToken": result.MFAToken})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  result.Tokens.AccessToken,
		"refreshToken": result.Tokens.RefreshToken,
		"expiresAt":    result.Tokens.ExpiresAt,
		"tokenType":    result.Tokens.TokenType,
		"user": gin.H{
			"id":            result.User.ID,
			"email":         result.User.Email,
			"firstName":     result.User.FirstName,
			"lastName":      result.User.LastName,
			"emailVerified": result.User.EmailVerified,
		},
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken, c.GetHeader("User-Agent"), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  tokens.AccessToken,
		"refreshToken": tokens.RefreshToken,
		"expiresAt":    tokens.ExpiresAt,
		"tokenType":    tokens.TokenType,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.LogoutSession(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
