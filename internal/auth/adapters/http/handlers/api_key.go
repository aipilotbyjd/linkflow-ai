package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/app/service"
)

type CreateAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

func (h *AuthHandler) ListAPIKeys(c *gin.Context) {
	userID := c.GetString("userID")
	workspaceID := c.GetHeader("X-Workspace-ID")

	keys, err := h.authService.ListAPIKeys(c.Request.Context(), userID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list api keys"})
		return
	}

	response := make([]gin.H, len(keys))
	for i, k := range keys {
		response[i] = gin.H{
			"id":         k.ID,
			"name":       k.Name,
			"keyPrefix":  k.KeyPrefix,
			"scopes":     k.Scopes,
			"lastUsedAt": k.LastUsedAt,
			"expiresAt":  k.ExpiresAt,
			"createdAt":  k.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"keys": response})
}

func (h *AuthHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	workspaceID := c.GetHeader("X-Workspace-ID")

	if len(req.Scopes) == 0 {
		req.Scopes = []string{"workflows:read", "workflows:execute"}
	}

	apiKey, rawKey, err := h.authService.CreateAPIKey(c.Request.Context(), service.CreateAPIKeyInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Scopes:      req.Scopes,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create api key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        apiKey.ID,
		"name":      apiKey.Name,
		"key":       rawKey,
		"keyPrefix": apiKey.KeyPrefix,
		"scopes":    apiKey.Scopes,
		"expiresAt": apiKey.ExpiresAt,
		"createdAt": apiKey.CreatedAt,
		"message":   "save this key now. you won't be able to see it again.",
	})
}

func (h *AuthHandler) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	userID := c.GetString("userID")

	if err := h.authService.RevokeAPIKey(c.Request.Context(), keyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "api key revoked"})
}
