package detector

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Service represents an IP detection service with IPv4 and IPv6 endpoints
type Service struct {
	Name    string
	IPv4URL string
	IPv6URL string
}

// Available IP detection services with dedicated IPv4/IPv6 endpoints
var Services = []Service{
	{
		Name:    "ipify",
		IPv4URL: "https://api4.ipify.org",
		IPv6URL: "https://api6.ipify.org",
	},
	{
		Name:    "icanhazip.com",
		IPv4URL: "https://ipv4.icanhazip.com",
		IPv6URL: "https://ipv6.icanhazip.com",
	},
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

// fetchIP makes an HTTP request and returns the IP address
func fetchIP(url string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ip_detector/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch IP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	return ip, nil
}

// DetectIPv4 fetches the public IPv4 address from the specified service
func DetectIPv4(service *Service) (string, error) {
	return fetchIP(service.IPv4URL)
}

// DetectIPv6 fetches the public IPv6 address from the specified service
func DetectIPv6(service *Service) (string, error) {
	return fetchIP(service.IPv6URL)
}

// DetectIPv4WithFallback tries the primary service first, then falls back to others
func DetectIPv4WithFallback(primaryService string) (string, string, error) {
	// Try primary service first
	primary := GetServiceByName(primaryService)
	if primary != nil {
		ip, err := DetectIPv4(primary)
		if err == nil && ip != "" {
			return ip, primary.Name, nil
		}
	}

	// Fallback to other services
	for _, service := range Services {
		if service.Name == primaryService {
			continue
		}
		ip, err := DetectIPv4(&service)
		if err == nil && ip != "" {
			return ip, service.Name, nil
		}
	}

	return "", "", fmt.Errorf("all IPv4 detection services failed")
}

// DetectIPv6WithFallback tries the primary service first, then falls back to others
// Returns empty string without error if IPv6 is not available
func DetectIPv6WithFallback(primaryService string) (string, string, error) {
	// Try primary service first
	primary := GetServiceByName(primaryService)
	if primary != nil {
		ip, err := DetectIPv6(primary)
		if err == nil && ip != "" {
			return ip, primary.Name, nil
		}
	}

	// Fallback to other services
	for _, service := range Services {
		if service.Name == primaryService {
			continue
		}
		ip, err := DetectIPv6(&service)
		if err == nil && ip != "" {
			return ip, service.Name, nil
		}
	}

	// IPv6 not available is not an error, just return empty
	return "", "", nil
}

// DetectIPWithFallback (legacy) - detects IPv4 for backward compatibility
func DetectIPWithFallback(primaryService string) (string, string, error) {
	return DetectIPv4WithFallback(primaryService)
}
