package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ip_detector/config"
	"ip_detector/detector"
	"ip_detector/notifier"
)

func main() {
	// Command line flags
	checkOnly := flag.Bool("check", false, "Check and display current IP without notifications")
	testNotify := flag.Bool("test-notify", false, "Send a test notification to Telegram")
	reconfigure := flag.Bool("reconfigure", false, "Reconfigure the application")
	daemon := flag.Bool("daemon", false, "Run in daemon mode (check IP periodically)")
	interval := flag.Int("interval", 300, "Check interval in seconds for daemon mode (default: 300)")
	flag.Parse()

	// Handle check-only mode (works without configuration)
	if *checkOnly {
		service := "ipify"
		if config.Exists() {
			cfg, err := config.Load()
			if err == nil {
				service = cfg.SelectedService
			}
		}
		ip, usedService, err := detector.DetectIPWithFallback(service)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to detect IP: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Current IP: %s (via %s)\n", ip, usedService)
		return
	}

	// Check if first run or reconfigure requested
	if !config.Exists() || *reconfigure {
		if err := runSetupWizard(); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n✅ Configuration saved successfully!")
		if *reconfigure {
			return
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Handle test notification
	if *testNotify {
		if err := sendTestNotification(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send test notification: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Test notification sent successfully!")
		return
	}

	// Handle daemon mode
	if *daemon {
		runDaemon(cfg, *interval)
		return
	}

	// Default: single check with notification
	if err := checkAndNotify(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSetupWizard() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║       IP Detector Setup Wizard         ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// Select IP detection service
	fmt.Println("Available IP detection services:")
	for i, service := range detector.Services {
		fmt.Printf("  %d. %s (%s)\n", i+1, service.Name, service.URL)
	}
	fmt.Print("\nSelect a service (1-5): ")
	serviceInput, _ := reader.ReadString('\n')
	serviceIdx, err := strconv.Atoi(strings.TrimSpace(serviceInput))
	if err != nil || serviceIdx < 1 || serviceIdx > len(detector.Services) {
		return fmt.Errorf("invalid service selection")
	}
	selectedService := detector.Services[serviceIdx-1].Name

	fmt.Println()
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Telegram Configuration")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println()

	// Get Telegram Bot Token
	fmt.Print("Enter your Telegram Bot Token: ")
	botToken, _ := reader.ReadString('\n')
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return fmt.Errorf("bot token cannot be empty")
	}

	// Get Telegram Chat ID
	fmt.Print("Enter your Telegram Chat ID: ")
	chatID, _ := reader.ReadString('\n')
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat ID cannot be empty")
	}

	// Create and save configuration
	_, err = config.CreateNew(selectedService, botToken, chatID)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// Test the configuration
	fmt.Println("\n🔄 Testing Telegram connection...")
	cfg, _ := config.Load()
	if err := sendTestNotification(cfg); err != nil {
		fmt.Printf("⚠️  Warning: Test notification failed: %v\n", err)
		fmt.Print("Continue anyway? (y/n): ")
		confirm, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			return fmt.Errorf("setup cancelled")
		}
	} else {
		fmt.Println("✅ Test notification sent successfully!")
	}

	return nil
}

func sendTestNotification(cfg *config.Config) error {
	botToken, err := cfg.GetBotToken()
	if err != nil {
		return fmt.Errorf("failed to decrypt bot token: %w", err)
	}

	chatID, err := cfg.GetChatID()
	if err != nil {
		return fmt.Errorf("failed to decrypt chat ID: %w", err)
	}

	tn := notifier.NewTelegramNotifier(botToken, chatID)
	return tn.SendTestNotification()
}

func checkAndNotify(cfg *config.Config) error {
	// Detect current IP
	ip, service, err := detector.DetectIPWithFallback(cfg.SelectedService)
	if err != nil {
		return fmt.Errorf("failed to detect IP: %w", err)
	}

	fmt.Printf("Current IP: %s (via %s)\n", ip, service)

	// Check if IP has changed
	if ip == cfg.LastKnownIP {
		fmt.Println("No IP change detected.")
		return nil
	}

	oldIP := cfg.LastKnownIP
	now := time.Now()

	// Update configuration
	cfg.LastKnownIP = ip
	cfg.LastChecked = now.Format(time.RFC3339)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Add to history
	if err := config.AddHistoryEntry(oldIP, ip); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save history: %v\n", err)
	}

	// Send notification
	botToken, err := cfg.GetBotToken()
	if err != nil {
		return fmt.Errorf("failed to decrypt bot token: %w", err)
	}

	chatID, err := cfg.GetChatID()
	if err != nil {
		return fmt.Errorf("failed to decrypt chat ID: %w", err)
	}

	tn := notifier.NewTelegramNotifier(botToken, chatID)
	if err := tn.SendIPChangeNotification(oldIP, ip, now); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	if oldIP == "" {
		fmt.Println("✅ Initial IP recorded and notification sent.")
	} else {
		fmt.Printf("✅ IP changed from %s to %s. Notification sent.\n", oldIP, ip)
	}

	return nil
}

func runDaemon(cfg *config.Config, intervalSeconds int) {
	fmt.Printf("Starting IP detector daemon (checking every %d seconds)...\n", intervalSeconds)
	fmt.Println("Press Ctrl+C to stop.")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	if err := checkAndNotify(cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	for {
		select {
		case <-sigChan:
			fmt.Println("\nReceived shutdown signal. Exiting gracefully...")
			return
		case <-ticker.C:
			// Reload config in case it was modified
			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				continue
			}
			if err := checkAndNotify(cfg); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}
