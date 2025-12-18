// Package nodes provides Discord node implementation
package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

func init() {
	runtime.Register(&DiscordNode{})
}

// DiscordNode implements Discord operations
type DiscordNode struct {
	client *http.Client
}

func (n *DiscordNode) GetType() string { return "discord" }
func (n *DiscordNode) Validate(config map[string]interface{}) error { return nil }

func (n *DiscordNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "discord",
		Name:        "Discord",
		Description: "Send messages and interact with Discord servers",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "discord",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Send Message", Value: "sendMessage"}, {Label: "Send Webhook", Value: "sendWebhook"},
				{Label: "Edit Message", Value: "editMessage"}, {Label: "Delete Message", Value: "deleteMessage"},
			}},
			{Name: "webhookUrl", Type: "string"},
			{Name: "channelId", Type: "string"},
			{Name: "messageId", Type: "string"},
			{Name: "content", Type: "string"},
			{Name: "embeds", Type: "json"},
			{Name: "username", Type: "string"},
		},
	}
}

func (n *DiscordNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 30 * time.Second}
	}

	operation, _ := input.NodeConfig["operation"].(string)

	// Handle webhook separately (no auth required)
	if operation == "sendWebhook" {
		return n.sendWebhook(ctx, input)
	}

	accessToken, _ := input.Credentials["access_token"].(string)
	botToken, _ := input.Credentials["bot_token"].(string)
	token := accessToken
	tokenType := "Bearer"
	if botToken != "" {
		token = botToken
		tokenType = "Bot"
	}

	if token == "" {
		return nil, fmt.Errorf("access_token or bot_token required")
	}

	baseURL := "https://discord.com/api/v10"
	headers := map[string]string{
		"Authorization": fmt.Sprintf("%s %s", tokenType, token),
		"Content-Type":  "application/json",
	}

	var result map[string]interface{}
	var err error

	channelID, _ := input.NodeConfig["channelId"].(string)
	guildID, _ := input.NodeConfig["guildId"].(string)
	messageID, _ := input.NodeConfig["messageId"].(string)

	switch operation {
	case "sendMessage":
		body := map[string]interface{}{
			"content": input.NodeConfig["content"],
		}
		if embeds, ok := input.NodeConfig["embeds"].([]interface{}); ok {
			body["embeds"] = embeds
		}
		if tts, ok := input.NodeConfig["tts"].(bool); ok {
			body["tts"] = tts
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/channels/%s/messages", baseURL, channelID), body, headers)

	case "editMessage":
		body := map[string]interface{}{
			"content": input.NodeConfig["content"],
		}
		if embeds, ok := input.NodeConfig["embeds"].([]interface{}); ok {
			body["embeds"] = embeds
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/channels/%s/messages/%s", baseURL, channelID, messageID), body, headers)

	case "deleteMessage":
		result, err = n.doRequest(ctx, "DELETE", fmt.Sprintf("%s/channels/%s/messages/%s", baseURL, channelID, messageID), nil, headers)

	case "getChannel":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/channels/%s", baseURL, channelID), nil, headers)

	case "listChannels":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/guilds/%s/channels", baseURL, guildID), nil, headers)

	case "getGuild":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/guilds/%s", baseURL, guildID), nil, headers)

	case "listGuilds":
		result, err = n.doRequest(ctx, "GET", baseURL+"/users/@me/guilds", nil, headers)

	case "getUser":
		userID, _ := input.NodeConfig["userId"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/users/%s", baseURL, userID), nil, headers)

	case "listMembers":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/guilds/%s/members?limit=100", baseURL, guildID), nil, headers)

	case "addReaction":
		emoji, _ := input.NodeConfig["emoji"].(string)
		result, err = n.doRequest(ctx, "PUT", fmt.Sprintf("%s/channels/%s/messages/%s/reactions/%s/@me", baseURL, channelID, messageID, emoji), nil, headers)

	case "removeReaction":
		emoji, _ := input.NodeConfig["emoji"].(string)
		result, err = n.doRequest(ctx, "DELETE", fmt.Sprintf("%s/channels/%s/messages/%s/reactions/%s/@me", baseURL, channelID, messageID, emoji), nil, headers)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *DiscordNode) sendWebhook(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	webhookURL, _ := input.NodeConfig["webhookUrl"].(string)
	if webhookURL == "" {
		return nil, fmt.Errorf("webhookUrl required")
	}

	body := map[string]interface{}{
		"content": input.NodeConfig["content"],
	}
	if username, ok := input.NodeConfig["username"].(string); ok && username != "" {
		body["username"] = username
	}
	if avatarUrl, ok := input.NodeConfig["avatarUrl"].(string); ok && avatarUrl != "" {
		body["avatar_url"] = avatarUrl
	}
	if embeds, ok := input.NodeConfig["embeds"].([]interface{}); ok {
		body["embeds"] = embeds
	}
	if tts, ok := input.NodeConfig["tts"].(bool); ok {
		body["tts"] = tts
	}

	result, err := n.doRequest(ctx, "POST", webhookURL, body, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *DiscordNode) doRequest(ctx context.Context, method, urlStr string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed: %s", string(data))
	}

	// Handle empty response
	if len(data) == 0 {
		return map[string]interface{}{"success": true}, nil
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}
