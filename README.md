# ACL Engine

A high-performance Access Control List (ACL) engine for Go, extracted from [Hysteria](https://github.com/apernet/hysteria). It provides flexible rule-based routing with support for IP, CIDR, domain matching, and GeoIP/GeoSite databases.

## Features

- **Multiple matching strategies**: IP, CIDR, domain (exact/wildcard/suffix)
- **GeoIP/GeoSite support**: Compatible with v2ray geo data format with auto-download
- **Protocol & port filtering**: TCP/UDP with port ranges
- **IP hijacking**: Redirect matched traffic to different IPs
- **LRU caching**: High-performance match result caching
- **Generic outbound type**: Use any type as your outbound identifier

## Installation

```bash
go get github.com/xflash-panda/acl-engine
```

## Quick Start

```go
package main

import (
    "fmt"
    "net"

    "github.com/xflash-panda/acl-engine/pkg/acl"
)

func main() {
    // Define ACL rules
    text := `
# Comments start with #
direct(1.1.1.1)
direct(192.168.0.0/16)
proxy(*.google.com)
proxy(geosite:youtube)
reject(geoip:cn, udp/443)
block(all, tcp/22)
`

    // Parse rules
    rules, err := acl.ParseTextRules(text)
    if err != nil {
        panic(err)
    }

    // Define outbounds
    outbounds := map[string]string{
        "direct": "DIRECT",
        "proxy":  "PROXY",
        "reject": "REJECT",
        "block":  "BLOCK",
    }

    // Create GeoLoader with auto-download (recommended)
    geoLoader := acl.NewAutoGeoLoader("./data")

    // Compile rules
    compiled, err := acl.Compile(rules, outbounds, 1024, geoLoader)
    if err != nil {
        panic(err)
    }

    // Match traffic
    host := acl.HostInfo{
        Name: "www.google.com",
        IPv4: net.ParseIP("142.250.80.46"),
    }
    outbound, hijackIP := compiled.Match(host, acl.ProtocolTCP, 443)
    fmt.Printf("Outbound: %s, HijackIP: %v\n", outbound, hijackIP)
}
```

## Rule Syntax

```
outbound(address[, protoPort][, hijackAddress])
```

### Address Types

| Type | Example | Description |
|------|---------|-------------|
| IP | `1.2.3.4` | Single IP address |
| CIDR | `192.168.0.0/16` | IP range in CIDR notation |
| Domain | `example.com` | Exact domain match |
| Wildcard | `*.example.com` | Wildcard domain match |
| Suffix | `suffix:example.com` | Domain suffix match |
| GeoIP | `geoip:cn` | Country code from GeoIP database |
| GeoSite | `geosite:google` | Site list from GeoSite database |
| GeoSite with attr | `geosite:google@cn` | GeoSite with attributes filter |
| All | `all` or `*` | Match everything |

### Protocol & Port

| Format | Description |
|--------|-------------|
| *(empty)* | All protocols and ports |
| `*` | All protocols and ports |
| `tcp` | TCP only, all ports |
| `udp` | UDP only, all ports |
| `tcp/443` | TCP port 443 |
| `udp/53` | UDP port 53 |
| `*/443` | Both TCP and UDP, port 443 |
| `tcp/8000-9000` | TCP port range 8000-9000 |

### Examples

```
# Direct connection for private networks
direct(192.168.0.0/16)
direct(10.0.0.0/8)

# Proxy for specific domains
proxy(*.google.com)
proxy(suffix:youtube.com)

# Use GeoIP/GeoSite
proxy(geoip:us)
proxy(geosite:netflix)
proxy(geosite:google@cn)

# Block specific traffic
reject(all, udp/443)  # Block QUIC

# Hijack DNS to local resolver
direct(all, udp/53, 127.0.0.1)

# Default rule (should be last)
proxy(all)
```

## GeoIP/GeoSite Usage

The library provides three GeoLoader implementations:

### 1. AutoGeoLoader (Recommended)

Automatically downloads and updates geo data files from CDN:

```go
// Basic usage - auto download to ./data directory
geoLoader := acl.NewAutoGeoLoader("./data")

// Advanced configuration
geoLoader := &acl.AutoGeoLoader{
    DataDir:        "./data",                    // Directory to store files
    UpdateInterval: 7 * 24 * time.Hour,          // Update interval (default: 7 days)
    Logger: func(format string, args ...interface{}) {
        log.Printf(format, args...)              // Optional: log download progress
    },
}

// Custom download URLs (e.g., use mirror)
geoLoader := &acl.AutoGeoLoader{
    DataDir:    "./data",
    GeoIPURL:   "https://your-mirror.com/geoip.dat",
    GeoSiteURL: "https://your-mirror.com/geosite.dat",
}
```

**Features:**
- Auto downloads `geoip.dat` and `geosite.dat` if not exist
- Auto updates files every 7 days (configurable)
- Falls back to existing files if download fails
- Downloads from CDN: `cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/`

### 2. FileGeoLoader

Load from local files only (no auto-download):

```go
geoLoader := acl.NewFileGeoLoader("./geoip.dat", "./geosite.dat")
```

### 3. NilGeoLoader

For rules that don't use GeoIP/GeoSite:

```go
geoLoader := &acl.NilGeoLoader{}
```

### Manual Download

If you prefer to download geo data manually:

```bash
# Download from Loyalsoldier (recommended, updated daily)
wget https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/geoip.dat
wget https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/geosite.dat

# Or from v2fly official
wget https://github.com/v2fly/geoip/releases/latest/download/geoip.dat
wget https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat -O geosite.dat
```

### GeoIP Country Codes

Common country codes for `geoip:` rules:

| Code | Country |
|------|---------|
| `cn` | China |
| `us` | United States |
| `jp` | Japan |
| `hk` | Hong Kong |
| `tw` | Taiwan |
| `sg` | Singapore |
| `private` | Private/LAN IPs |

### GeoSite Categories

Common categories for `geosite:` rules:

| Category | Description |
|----------|-------------|
| `google` | Google services |
| `youtube` | YouTube |
| `facebook` | Facebook/Meta |
| `twitter` | Twitter/X |
| `netflix` | Netflix |
| `telegram` | Telegram |
| `cn` | China mainland sites |
| `geolocation-cn` | Sites with servers in China |
| `geolocation-!cn` | Sites without servers in China |
| `category-ads` | Ad domains |
| `category-ads-all` | All ad domains |

**With attributes:**
```
# Only match Google domains that have @cn attribute
proxy(geosite:google@cn)
```

## API Reference

### Types

```go
// HostInfo contains information about the target host
type HostInfo struct {
    Name string   // Domain name (optional)
    IPv4 net.IP   // IPv4 address (optional)
    IPv6 net.IP   // IPv6 address (optional)
}

// Protocol type
type Protocol int
const (
    ProtocolBoth Protocol = iota
    ProtocolTCP
    ProtocolUDP
)

// GeoLoader interface for loading geo databases
type GeoLoader interface {
    LoadGeoIP() (map[string]*v2geo.GeoIP, error)
    LoadGeoSite() (map[string]*v2geo.GeoSite, error)
}
```

### Functions

```go
// Parse ACL rules from text
func ParseTextRules(text string) ([]TextRule, error)

// Compile rules into a matcher
func Compile[O Outbound](
    rules []TextRule,
    outbounds map[string]O,
    cacheSize int,
    geoLoader GeoLoader,
) (CompiledRuleSet[O], error)

// Create GeoLoaders
func NewAutoGeoLoader(dataDir string) *AutoGeoLoader  // With auto-download
func NewFileGeoLoader(geoIPPath, geoSitePath string) *FileGeoLoader  // File only
```

### CompiledRuleSet

```go
type CompiledRuleSet[O Outbound] interface {
    // Match returns the outbound and optional hijack IP for the given host
    Match(host HostInfo, proto Protocol, port uint16) (O, net.IP)
}
```

## License

MIT License
