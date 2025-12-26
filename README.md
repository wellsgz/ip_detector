# IP Detector

A CLI application that monitors your public IP address and sends Telegram notifications when it changes. Perfect for cloud VMs using NAT networks where upstream IPs may change.

## Features

- **5 IP Detection Services**: ipify, ifconfig.me, ipinfo.io, api.ip.sb, icanhazip.com
- **Automatic Fallback**: If primary service fails, automatically tries others
- **Telegram Notifications**: Get notified when your IP changes
- **Secure Storage**: Credentials encrypted with AES-256-GCM
- **IP History**: Keeps last 500 IP changes in JSON format
- **Daemon Mode**: Continuously monitor IP at configurable intervals

## Installation

Download the latest binary for your platform from [Releases](../../releases):
- `ip_detector-linux-x86_64` - Linux (Intel/AMD)
- `ip_detector-linux-arm64` - Linux (ARM64)
- `ip_detector-darwin-x86_64` - macOS (Intel)
- `ip_detector-darwin-arm64` - macOS (Apple Silicon)

```bash
# Example for Linux x86_64
chmod +x ip_detector-linux-x86_64
./ip_detector-linux-x86_64
```

## Usage

### First Run

On first run, the app will guide you through setup:

```bash
./ip_detector
```

You'll be prompted to:
1. Select an IP detection service
2. Enter your Telegram Bot Token
3. Enter your Telegram Chat ID

### Commands

```bash
# Check current IP (no notification)
./ip_detector --check

# Check IP and send notification if changed
./ip_detector

# Send test notification
./ip_detector --test-notify

# Run in daemon mode (check every 5 minutes)
./ip_detector --daemon

# Run in daemon mode with custom interval (in seconds)
./ip_detector --daemon --interval 60

# Reconfigure the application
./ip_detector --reconfigure
```

## Configuration

Configuration is stored in `~/.ip_detector/`:
- `config.json` - Encrypted credentials and settings
- `ip_history.json` - Last 500 IP changes

## Running as a Service

### systemd (Ubuntu, Debian, CentOS, etc.)

```bash
# Copy binary and service file
sudo cp ip_detector /usr/local/bin/
sudo cp contrib/ip_detector@.service /etc/systemd/system/

# Enable and start for current user
sudo systemctl enable ip_detector@$USER
sudo systemctl start ip_detector@$USER

# Check status
sudo systemctl status ip_detector@$USER
```

### OpenRC (Alpine, Gentoo)

```bash
# Copy binary and init script
sudo cp ip_detector /usr/local/bin/
sudo cp contrib/ip_detector.openrc /etc/init.d/ip_detector.$USER
sudo chmod +x /etc/init.d/ip_detector.$USER

# Enable and start
sudo rc-update add ip_detector.$USER default
sudo rc-service ip_detector.$USER start
```

## Getting Telegram Credentials

1. **Create a Bot**: Message [@BotFather](https://t.me/BotFather) on Telegram
   - Send `/newbot` and follow instructions
   - Copy the bot token provided

2. **Get Chat ID**: 
   - Message your new bot
   - Visit `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
   - Find your `chat.id` in the response

## License

MIT
