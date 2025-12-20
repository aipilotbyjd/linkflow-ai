package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) OAuthGoogle(c *gin.Context) {
	// TODO: Implement Google OAuth
	c.JSON(http.StatusNotImplemented, gin.H{"error": "google oauth not implemented"})
}

func (h *AuthHandler) OAuthGitHub(c *gin.Context) {
	// TODO: Implement GitHub OAuth
	c.JSON(http.StatusNotImplemented, gin.H{"error": "github oauth not implemented"})
}

func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	// TODO: Implement OAuth callback
	c.JSON(http.StatusNotImplemented, gin.H{"error": "oauth callback not implemented"})
}
