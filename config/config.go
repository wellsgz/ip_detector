package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ip_detector/storage"
)

const (
	configDir     = ".ip_detector"
	configFile    = "config.json"
	historyFile   = "ip_history.json"
	maxHistoryLen = 500
)

// Config represents the application configuration
type Config struct {
	SelectedService   string `json:"selected_service"`
	EncryptedBotToken string `json:"encrypted_bot_token"`
	EncryptedChatID   string `json:"encrypted_chat_id"`
	LastKnownIPv4     string `json:"last_known_ipv4"`
	LastKnownIPv6     string `json:"last_known_ipv6"`
	LastChecked       string `json:"last_checked"`
	// Legacy field for backward compatibility (will be migrated to LastKnownIPv4)
	LastKnownIP string `json:"last_known_ip,omitempty"`
}

// IPHistoryEntry represents a single IP change record
type IPHistoryEntry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"` // "ipv4" or "ipv6"
	OldIP     string `json:"old_ip"`
	NewIP     string `json:"new_ip"`
}

// getConfigDir returns the path to the config directory
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDir), nil
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

// getHistoryPath returns the path to the history file
func getHistoryPath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, historyFile), nil
}

// Exists checks if the config file exists
func Exists() bool {
	path, err := getConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Migrate legacy LastKnownIP to LastKnownIPv4
	if config.LastKnownIP != "" && config.LastKnownIPv4 == "" {
		config.LastKnownIPv4 = config.LastKnownIP
		config.LastKnownIP = ""
		// Save the migrated config
		_ = config.Save()
	}

	return &config, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	dir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := getConfigPath()
	if err != nil {
		return err
	}

	// Clear legacy field
	c.LastKnownIP = ""

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetBotToken decrypts and returns the bot token
func (c *Config) GetBotToken() (string, error) {
	return storage.Decrypt(c.EncryptedBotToken)
}

// GetChatID decrypts and returns the chat ID
func (c *Config) GetChatID() (string, error) {
	return storage.Decrypt(c.EncryptedChatID)
}

// SetCredentials encrypts and stores the Telegram credentials
func (c *Config) SetCredentials(botToken, chatID string) error {
	encryptedToken, err := storage.Encrypt(botToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt bot token: %w", err)
	}

	encryptedChatID, err := storage.Encrypt(chatID)
	if err != nil {
		return fmt.Errorf("failed to encrypt chat ID: %w", err)
	}

	c.EncryptedBotToken = encryptedToken
	c.EncryptedChatID = encryptedChatID
	return nil
}

// LoadHistory loads the IP history from disk
func LoadHistory() ([]IPHistoryEntry, error) {
	path, err := getHistoryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []IPHistoryEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var history []IPHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	return history, nil
}

// SaveHistory saves the IP history to disk
func SaveHistory(history []IPHistoryEntry) error {
	dir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := getHistoryPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize history: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// AddHistoryEntry adds a new entry to the IP history
func AddHistoryEntry(ipType, oldIP, newIP string) error {
	history, err := LoadHistory()
	if err != nil {
		return err
	}

	entry := IPHistoryEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      ipType,
		OldIP:     oldIP,
		NewIP:     newIP,
	}

	// Prepend new entry
	history = append([]IPHistoryEntry{entry}, history...)

	// Trim to max length
	if len(history) > maxHistoryLen {
		history = history[:maxHistoryLen]
	}

	return SaveHistory(history)
}

// CreateNew creates a new configuration with the given settings
func CreateNew(service, botToken, chatID string) (*Config, error) {
	config := &Config{
		SelectedService: service,
	}

	if err := config.SetCredentials(botToken, chatID); err != nil {
		return nil, err
	}

	if err := config.Save(); err != nil {
		return nil, err
	}

	return config, nil
}
