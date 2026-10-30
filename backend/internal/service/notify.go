package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ChannelConfig 通知渠道配置（解密后的 JSON）
type ChannelConfig struct {
	// email
	To string `json:"to"` // 多个用逗号分隔
	// webhook / wecom
	URL string `json:"url"`
}

// SendToChannel 按渠道类型发送通知
func SendToChannel(chType string, cfg ChannelConfig, title, content string) error {
	switch chType {
	case "email":
		return nil // email 渠道由 handler/scheduler 持有 Mailer 发送，见 SendEmailChannel
	case "webhook":
		return sendWebhook(cfg.URL, map[string]any{
			"title":   title,
			"content": content,
			"time":    time.Now().Format(time.RFC3339),
		})
	case "wecom":
		// 企业微信群机器人
		return sendWebhook(cfg.URL, map[string]any{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": fmt.Sprintf("**%s**\n%s", title, content),
			},
		})
	default:
		return fmt.Errorf("unknown channel type: %s", chType)
	}
}

func sendWebhook(url string, payload any) error {
	if url == "" {
		return fmt.Errorf("webhook url is empty")
	}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}
