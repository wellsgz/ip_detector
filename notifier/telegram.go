package notifier

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TelegramNotifier handles sending notifications via Telegram
type TelegramNotifier struct {
	BotToken string
	ChatID   string
}

// NewTelegramNotifier creates a new TelegramNotifier
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: botToken,
		ChatID:   chatID,
	}
}

// SendMessage sends a message via Telegram Bot API
func (t *TelegramNotifier) SendMessage(message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.PostForm(apiURL, url.Values{
		"chat_id":    {t.ChatID},
		"text":       {message},
		"parse_mode": {"Markdown"},
	})
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendIPChangeNotification sends a formatted IP change notification with hostname
func (t *TelegramNotifier) SendIPChangeNotification(hostname, ipType, oldIP, newIP string, timestamp time.Time) error {
	var message string
	typeLabel := "IPv4"
	if ipType == "ipv6" {
		typeLabel = "IPv6"
	}

	if oldIP == "" {
		message = fmt.Sprintf("🌐 *IP Detector Initialized*\n\n"+
			"🖥️ Host: `%s`\n"+
			"🏷️ Type: %s\n"+
			"📍 Current IP: `%s`\n"+
			"🕐 Time: %s",
			hostname, typeLabel, newIP, timestamp.Format("2006-01-02 15:04:05 MST"))
	} else {
		message = fmt.Sprintf("🔄 *IP Address Changed*\n\n"+
			"🖥️ Host: `%s`\n"+
			"🏷️ Type: %s\n"+
			"📍 Old IP: `%s`\n"+
			"📍 New IP: `%s`\n"+
			"🕐 Time: %s",
			hostname, typeLabel, oldIP, newIP, timestamp.Format("2006-01-02 15:04:05 MST"))
	}

	return t.SendMessage(message)
}

// SendTestNotification sends a test notification with hostname
func (t *TelegramNotifier) SendTestNotification(hostname string) error {
	message := fmt.Sprintf("✅ *IP Detector Test*\n\n"+
		"🖥️ Host: `%s`\n"+
		"Telegram notification is working correctly!", hostname)
	return t.SendMessage(message)
}
