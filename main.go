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

	// Get hostname for notifications
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// Handle check-only mode (works without configuration)
	if *checkOnly {
		service := "ipify"
		if config.Exists() {
			cfg, err := config.Load()
			if err == nil {
				service = cfg.SelectedService
			}
		}

		fmt.Println("Detecting IP addresses...")
		fmt.Printf("Hostname: %s\n\n", hostname)

		// Detect IPv4
		ipv4, v4Service, err := detector.DetectIPv4WithFallback(service)
		if err != nil {
			fmt.Printf("IPv4: Not detected (%v)\n", err)
		} else {
			fmt.Printf("IPv4: %s (via %s)\n", ipv4, v4Service)
		}

		// Detect IPv6
		ipv6, v6Service, _ := detector.DetectIPv6WithFallback(service)
		if ipv6 == "" {
			fmt.Println("IPv6: Not available")
		} else {
			fmt.Printf("IPv6: %s (via %s)\n", ipv6, v6Service)
		}
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
		if err := sendTestNotification(cfg, hostname); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send test notification: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Test notification sent successfully!")
		return
	}

	// Handle daemon mode
	if *daemon {
		runDaemon(cfg, hostname, *interval)
		return
	}

	// Default: single check with notification
	if err := checkAndNotify(cfg, hostname); err != nil {
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
		fmt.Printf("  %d. %s\n", i+1, service.Name)
	}
	fmt.Print("\nSelect a service (1-2): ")
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
	hostname, _ := os.Hostname()
	if err := sendTestNotification(cfg, hostname); err != nil {
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

func sendTestNotification(cfg *config.Config, hostname string) error {
	botToken, err := cfg.GetBotToken()
	if err != nil {
		return fmt.Errorf("failed to decrypt bot token: %w", err)
	}

	chatID, err := cfg.GetChatID()
	if err != nil {
		return fmt.Errorf("failed to decrypt chat ID: %w", err)
	}

	tn := notifier.NewTelegramNotifier(botToken, chatID)
	return tn.SendTestNotification(hostname)
}

func checkAndNotify(cfg *config.Config, hostname string) error {
	now := time.Now()

	// Detect IPv4
	ipv4, v4Service, err := detector.DetectIPv4WithFallback(cfg.SelectedService)
	if err != nil {
		fmt.Printf("⚠️  IPv4 detection failed: %v\n", err)
		ipv4 = ""
	} else {
		fmt.Printf("IPv4: %s (via %s)\n", ipv4, v4Service)
	}

	// Detect IPv6
	ipv6, v6Service, _ := detector.DetectIPv6WithFallback(cfg.SelectedService)
	if ipv6 != "" {
		fmt.Printf("IPv6: %s (via %s)\n", ipv6, v6Service)
	} else {
		fmt.Println("IPv6: Not available")
	}

	// Check for changes
	ipv4Status := notifier.IPStatus{
		Current:  ipv4,
		Previous: cfg.LastKnownIPv4,
		Changed:  ipv4 != "" && ipv4 != cfg.LastKnownIPv4,
	}

	ipv6Status := notifier.IPStatus{
		Current:  ipv6,
		Previous: cfg.LastKnownIPv6,
		Changed:  ipv6 != "" && ipv6 != cfg.LastKnownIPv6,
	}

	// If anything changed, send notification
	if ipv4Status.Changed || ipv6Status.Changed {
		// Update config
		if ipv4 != "" {
			cfg.LastKnownIPv4 = ipv4
		}
		if ipv6 != "" {
			cfg.LastKnownIPv6 = ipv6
		}
		cfg.LastChecked = now.Format(time.RFC3339)

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		// Add history entries
		if ipv4Status.Changed {
			if err := config.AddHistoryEntry("ipv4", ipv4Status.Previous, ipv4); err != nil {
				fmt.Printf("⚠️  Warning: Failed to save IPv4 history: %v\n", err)
			}
		}
		if ipv6Status.Changed {
			if err := config.AddHistoryEntry("ipv6", ipv6Status.Previous, ipv6); err != nil {
				fmt.Printf("⚠️  Warning: Failed to save IPv6 history: %v\n", err)
			}
		}

		// Send combined notification
		botToken, err := cfg.GetBotToken()
		if err != nil {
			return fmt.Errorf("failed to decrypt bot token: %w", err)
		}

		chatID, err := cfg.GetChatID()
		if err != nil {
			return fmt.Errorf("failed to decrypt chat ID: %w", err)
		}

		tn := notifier.NewTelegramNotifier(botToken, chatID)
		if err := tn.SendCombinedIPNotification(hostname, ipv4Status, ipv6Status, now); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}

		fmt.Println("✅ Notification sent.")
	} else {
		fmt.Println("No IP changes detected.")
	}

	return nil
}

func runDaemon(cfg *config.Config, hostname string, intervalSeconds int) {
	fmt.Printf("Starting IP detector daemon (checking every %d seconds)...\n", intervalSeconds)
	fmt.Printf("Hostname: %s\n", hostname)
	fmt.Println("Press Ctrl+C to stop.")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	if err := checkAndNotify(cfg, hostname); err != nil {
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
			if err := checkAndNotify(cfg, hostname); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}
