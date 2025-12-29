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

// IPStatus holds the status of an IP (current value and whether it changed)
type IPStatus struct {
	Current  string
	Previous string
	Changed  bool
}

// SendCombinedIPNotification sends a notification with both IPv4 and IPv6 status
func (t *TelegramNotifier) SendCombinedIPNotification(hostname string, ipv4, ipv6 IPStatus, timestamp time.Time) error {
	var title string
	if (ipv4.Changed && ipv4.Previous == "") || (ipv6.Changed && ipv6.Previous == "") {
		title = "🌐 *IP Detector Initialized*"
	} else {
		title = "🔄 *IP Address Changed*"
	}

	// Build IPv4 section
	var ipv4Section string
	if ipv4.Current != "" {
		if ipv4.Changed {
			if ipv4.Previous == "" {
				ipv4Section = fmt.Sprintf("📍 IPv4: `%s` (new)", ipv4.Current)
			} else {
				ipv4Section = fmt.Sprintf("📍 IPv4: `%s` ← `%s`", ipv4.Current, ipv4.Previous)
			}
		} else {
			ipv4Section = fmt.Sprintf("📍 IPv4: `%s`", ipv4.Current)
		}
	} else {
		ipv4Section = "📍 IPv4: Not available"
	}

	// Build IPv6 section
	var ipv6Section string
	if ipv6.Current != "" {
		if ipv6.Changed {
			if ipv6.Previous == "" {
				ipv6Section = fmt.Sprintf("📍 IPv6: `%s` (new)", ipv6.Current)
			} else {
				ipv6Section = fmt.Sprintf("📍 IPv6: `%s` ← `%s`", ipv6.Current, ipv6.Previous)
			}
		} else {
			ipv6Section = fmt.Sprintf("📍 IPv6: `%s`", ipv6.Current)
		}
	} else {
		ipv6Section = "📍 IPv6: Not available"
	}

	message := fmt.Sprintf("%s\n\n"+
		"🖥️ Host: `%s`\n"+
		"%s\n"+
		"%s\n"+
		"🕐 Time: %s",
		title, hostname, ipv4Section, ipv6Section, timestamp.Format("2006-01-02 15:04:05 MST"))

	return t.SendMessage(message)
}

// SendTestNotification sends a test notification with hostname
func (t *TelegramNotifier) SendTestNotification(hostname string) error {
	message := fmt.Sprintf("✅ *IP Detector Test*\n\n"+
		"🖥️ Host: `%s`\n"+
		"Telegram notification is working correctly!", hostname)
	return t.SendMessage(message)
}
