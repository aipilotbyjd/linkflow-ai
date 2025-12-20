package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) MFASetup(c *gin.Context) {
	// TODO: Implement TOTP setup with github.com/pquerna/otp/totp
	c.JSON(http.StatusNotImplemented, gin.H{"error": "mfa setup not implemented"})
}

func (h *AuthHandler) MFAVerify(c *gin.Context) {
	// TODO: Implement TOTP verification
	c.JSON(http.StatusNotImplemented, gin.H{"error": "mfa verify not implemented"})
}

func (h *AuthHandler) MFADisable(c *gin.Context) {
	// TODO: Implement MFA disable
	c.JSON(http.StatusNotImplemented, gin.H{"error": "mfa disable not implemented"})
}
