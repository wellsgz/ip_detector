package detector

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Service represents an IP detection service
type Service struct {
	Name string
	URL  string
}

// Available IP detection services
var Services = []Service{
	{Name: "ipify", URL: "https://api.ipify.org"},
	{Name: "ifconfig.me", URL: "https://ifconfig.me/ip"},
	{Name: "ipinfo.io", URL: "https://ipinfo.io/ip"},
	{Name: "api.ip.sb", URL: "https://api.ip.sb/ip"},
	{Name: "icanhazip.com", URL: "https://icanhazip.com"},
}

// GetServiceByName returns a service by its name
func GetServiceByName(name string) *Service {
	for _, s := range Services {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

// DetectIP fetches the public IP address from the specified service
func DetectIP(service *Service) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", service.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "ip_detector/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch IP from %s: %w", service.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code from %s: %d", service.Name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	return ip, nil
}

// DetectIPWithFallback tries the primary service first, then falls back to others
func DetectIPWithFallback(primaryService string) (string, string, error) {
	// Try primary service first
	primary := GetServiceByName(primaryService)
	if primary != nil {
		ip, err := DetectIP(primary)
		if err == nil {
			return ip, primary.Name, nil
		}
	}

	// Fallback to other services
	for _, service := range Services {
		if service.Name == primaryService {
			continue
		}
		ip, err := DetectIP(&service)
		if err == nil {
			return ip, service.Name, nil
		}
	}

	return "", "", fmt.Errorf("all IP detection services failed")
}
