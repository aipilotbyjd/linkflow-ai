// Package nodes provides Telegram node implementation
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
	runtime.Register(&TelegramNode{})
}

// TelegramNode implements Telegram Bot API operations
type TelegramNode struct {
	client *http.Client
}

func (n *TelegramNode) GetType() string { return "telegram" }
func (n *TelegramNode) Validate(config map[string]interface{}) error { return nil }

func (n *TelegramNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "telegram",
		Name:        "Telegram",
		Description: "Send messages via Telegram Bot API",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "telegram",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Send Message", Value: "sendMessage"}, {Label: "Edit Message", Value: "editMessage"},
				{Label: "Delete Message", Value: "deleteMessage"}, {Label: "Send Photo", Value: "sendPhoto"},
			}},
			{Name: "chatId", Type: "string", Required: true},
			{Name: "messageId", Type: "number"},
			{Name: "text", Type: "string"},
			{Name: "parseMode", Type: "select", Default: "HTML", Options: []runtime.PropertyOption{
				{Label: "HTML", Value: "HTML"}, {Label: "Markdown", Value: "Markdown"},
			}},
			{Name: "fileUrl", Type: "string"},
			{Name: "caption", Type: "string"},
		},
	}
}

func (n *TelegramNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 30 * time.Second}
	}

	operation, _ := input.NodeConfig["operation"].(string)
	botToken, _ := input.Credentials["bot_token"].(string)
	if botToken == "" {
		botToken, _ = input.Credentials["token"].(string)
	}

	if botToken == "" {
		return nil, fmt.Errorf("bot_token required")
	}

	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s", botToken)
	chatID, _ := input.NodeConfig["chatId"].(string)

	var result map[string]interface{}
	var err error

	switch operation {
	case "sendMessage":
		body := map[string]interface{}{
			"chat_id": chatID,
			"text":    input.NodeConfig["text"],
		}
		if parseMode, ok := input.NodeConfig["parseMode"].(string); ok {
			body["parse_mode"] = parseMode
		}
		if disable, ok := input.NodeConfig["disableNotification"].(bool); ok {
			body["disable_notification"] = disable
		}
		if replyTo, ok := input.NodeConfig["replyToMessageId"].(float64); ok {
			body["reply_to_message_id"] = int(replyTo)
		}
		if markup, ok := input.NodeConfig["replyMarkup"].(map[string]interface{}); ok {
			body["reply_markup"] = markup
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/sendMessage", body)

	case "editMessage":
		messageID := int(input.NodeConfig["messageId"].(float64))
		body := map[string]interface{}{
			"chat_id":    chatID,
			"message_id": messageID,
			"text":       input.NodeConfig["text"],
		}
		if parseMode, ok := input.NodeConfig["parseMode"].(string); ok {
			body["parse_mode"] = parseMode
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/editMessageText", body)

	case "deleteMessage":
		messageID := int(input.NodeConfig["messageId"].(float64))
		body := map[string]interface{}{
			"chat_id":    chatID,
			"message_id": messageID,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/deleteMessage", body)

	case "sendPhoto":
		fileURL, _ := input.NodeConfig["fileUrl"].(string)
		body := map[string]interface{}{
			"chat_id": chatID,
			"photo":   fileURL,
		}
		if caption, ok := input.NodeConfig["caption"].(string); ok {
			body["caption"] = caption
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/sendPhoto", body)

	case "sendDocument":
		fileURL, _ := input.NodeConfig["fileUrl"].(string)
		body := map[string]interface{}{
			"chat_id":  chatID,
			"document": fileURL,
		}
		if caption, ok := input.NodeConfig["caption"].(string); ok {
			body["caption"] = caption
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/sendDocument", body)

	case "sendVideo":
		fileURL, _ := input.NodeConfig["fileUrl"].(string)
		body := map[string]interface{}{
			"chat_id": chatID,
			"video":   fileURL,
		}
		if caption, ok := input.NodeConfig["caption"].(string); ok {
			body["caption"] = caption
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/sendVideo", body)

	case "sendAudio":
		fileURL, _ := input.NodeConfig["fileUrl"].(string)
		body := map[string]interface{}{
			"chat_id": chatID,
			"audio":   fileURL,
		}
		if caption, ok := input.NodeConfig["caption"].(string); ok {
			body["caption"] = caption
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/sendAudio", body)

	case "getChat":
		body := map[string]interface{}{
			"chat_id": chatID,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/getChat", body)

	case "getChatMember":
		userID, _ := input.NodeConfig["userId"].(string)
		body := map[string]interface{}{
			"chat_id": chatID,
			"user_id": userID,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/getChatMember", body)

	case "getChatMembersCount":
		body := map[string]interface{}{
			"chat_id": chatID,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/getChatMembersCount", body)

	case "sendChatAction":
		action, _ := input.NodeConfig["action"].(string)
		body := map[string]interface{}{
			"chat_id": chatID,
			"action":  action,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/sendChatAction", body)

	case "getUpdates":
		result, err = n.doRequest(ctx, "POST", baseURL+"/getUpdates", nil)

	case "setWebhook":
		webhookURL, _ := input.NodeConfig["webhookUrl"].(string)
		body := map[string]interface{}{
			"url": webhookURL,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/setWebhook", body)

	case "deleteWebhook":
		result, err = n.doRequest(ctx, "POST", baseURL+"/deleteWebhook", nil)

	case "getMe":
		result, err = n.doRequest(ctx, "POST", baseURL+"/getMe", nil)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *TelegramNode) doRequest(ctx context.Context, method, urlStr string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if ok, _ := result["ok"].(bool); !ok {
		desc, _ := result["description"].(string)
		return nil, fmt.Errorf("telegram API error: %s", desc)
	}

	return result, nil
}
