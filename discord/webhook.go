package discord

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type WebhookExecutorConfig struct {
	webhookId    string
	webhookToken string
}

func NewWebhookExecutorConfig(webhookId string, webhookToken string) *WebhookExecutorConfig {
	return &WebhookExecutorConfig{webhookId: webhookId, webhookToken: webhookToken}
}

type WebhookExecutor struct {
	config *WebhookExecutorConfig
}

func NewWebhookExecutor(config *WebhookExecutorConfig) *WebhookExecutor {
	return &WebhookExecutor{config: config}
}

func (executor *WebhookExecutor) ExecuteWebhook(user string, content string, green bool) (err error) {
	var avatarUrl string
	if green {
		avatarUrl = "https://cdn.discordapp.com/attachments/628346028717768735/628593833336242186/pobrane.png"
	}

	body, err := json.Marshal(map[string]string{
		"content":    content,
		"username":   user,
		"avatar_url": avatarUrl,
	})
	if err != nil {
		return err
	}
	resp, err := http.Post("https://discordapp.com/api/webhooks/"+executor.config.webhookId+"/"+
		executor.config.webhookToken, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	return
}
