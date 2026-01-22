# ACL Engine

A high-performance Access Control List (ACL) engine for Go, extracted from [Hysteria](https://github.com/apernet/hysteria). It provides flexible rule-based routing with support for IP, CIDR, domain matching, GeoIP/GeoSite databases, and pluggable outbound connections.

## Features

- **Multiple matching strategies**: IP, CIDR, domain (exact/wildcard/suffix)
- **GeoIP/GeoSite support**: Multiple formats (DAT, MMDB, MetaDB, sing-geosite)
- **Protocol & port filtering**: TCP/UDP with port ranges
- **IP hijacking**: Redirect matched traffic to different IPs
- **LRU caching**: High-performance match result caching
- **Pluggable outbounds**: Direct, SOCKS5, HTTP proxy with TCP Fast Open support
- **DNS resolvers**: System, UDP, TCP, TLS (DoT), HTTPS (DoH)
- **ACL Router**: Combines ACL rules with outbounds for complete traffic routing

## Installation

```bash
go get github.com/xflash-panda/acl-engine
```

## Quick Start

### Using the Router (Recommended)

The Router combines ACL rules with outbounds for complete traffic routing:

```go
package main

import (
    "fmt"

    "github.com/xflash-panda/acl-engine/pkg/acl"
    "github.com/xflash-panda/acl-engine/pkg/outbound"
    "github.com/xflash-panda/acl-engine/pkg/resolver"
    "github.com/xflash-panda/acl-engine/pkg/router"
)

func main() {
    // Define ACL rules
    rules := `
# Direct for private networks
direct(192.168.0.0/16)
direct(10.0.0.0/8)

# Proxy for specific sites
proxy(*.google.com)
proxy(geosite:youtube)

# Reject QUIC
reject(all, udp/443)

# Default: direct
direct(all)
`

    // Create outbounds
    outbounds := []router.OutboundEntry{
        {"proxy", outbound.NewSOCKS5("127.0.0.1:1080", "", "")},
        // "direct" and "reject" are built-in
    }

    // Create GeoLoader
    geoLoader := &acl.AutoGeoLoader{
        DataDir:       "./data",
        GeoIPFormat:   acl.GeoIPFormatMMDB,
        GeoIPURL:      acl.MetaCubeXGeoIPMMDBURL,
        GeoSiteFormat: acl.GeoSiteFormatDAT,
        GeoSiteURL:    acl.MetaCubeXGeoSiteDatURL,
    }

    // Create router with custom DNS resolver
    r, err := router.New(
        rules,
        outbounds,
        geoLoader,
        router.WithResolver(resolver.NewUDP("8.8.8.8", 0)),
    )
    if err != nil {
        panic(err)
    }

    // Use the router
    conn, err := r.DialTCP(&outbound.Addr{Host: "www.google.com", Port: 443})
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    fmt.Println("Connected!")
}
```

### Using ACL Rules Only

If you only need ACL rule matching without the outbound implementations:

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
direct(1.1.1.1)
direct(192.168.0.0/16)
proxy(*.google.com)
reject(geoip:cn, udp/443)
`

    // Parse rules
    rules, err := acl.ParseTextRules(text)
    if err != nil {
        panic(err)
    }

    // Define outbounds (can be any type)
    outbounds := map[string]string{
        "direct": "DIRECT",
        "proxy":  "PROXY",
        "reject": "REJECT",
    }

    // Create GeoLoader
    geoLoader := &acl.AutoGeoLoader{
        DataDir:       "./data",
        GeoIPFormat:   acl.GeoIPFormatMMDB,
        GeoIPURL:      acl.MetaCubeXGeoIPMMDBURL,
        GeoSiteFormat: acl.GeoSiteFormatDAT,
        GeoSiteURL:    acl.MetaCubeXGeoSiteDatURL,
    }

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

## Package Overview

```
pkg/
├── acl/         # ACL rule parsing and compilation
├── outbound/    # Outbound connection implementations
├── resolver/    # DNS resolver implementations
└── router/      # ACL-based traffic router
```

### pkg/outbound

Provides pluggable outbound connection interfaces and implementations:

```go
import "github.com/xflash-panda/acl-engine/pkg/outbound"

// Interface
type Outbound interface {
    DialTCP(addr *Addr) (net.Conn, error)
    DialUDP(addr *Addr) (UDPConn, error)
}

// Address with optional DNS resolution info
type Addr struct {
    Host        string
    Port        uint16
    ResolveInfo *ResolveInfo
}

// Implementations
outbound.NewDirect(mode)                         // Direct connection
outbound.NewDirectWithOptions(opts)              // With TCP Fast Open, device binding
outbound.NewSOCKS5(addr, username, password)     // SOCKS5 proxy
outbound.NewHTTP(proxyURL, insecure)             // HTTP/HTTPS proxy
outbound.NewReject()                             // Reject all connections
```

#### Direct Outbound Modes

```go
outbound.DirectModeAuto  // Dual-stack "happy eyeballs"
outbound.DirectMode64    // Prefer IPv6, fallback to IPv4
outbound.DirectMode46    // Prefer IPv4, fallback to IPv6
outbound.DirectMode6     // IPv6 only
outbound.DirectMode4     // IPv4 only
```

#### Direct Outbound Options

```go
ob, _ := outbound.NewDirectWithOptions(outbound.DirectOptions{
    Mode:       outbound.DirectModeAuto,
    FastOpen:   true,              // TCP Fast Open
    DeviceName: "eth0",            // Bind to device (Linux only)
    BindIP4:    net.ParseIP("..."), // Bind to specific IPv4
    BindIP6:    net.ParseIP("..."), // Bind to specific IPv6
})
```

### pkg/resolver

Provides DNS resolver implementations:

```go
import "github.com/xflash-panda/acl-engine/pkg/resolver"

// Interface
type Resolver interface {
    Resolve(host string) (ipv4, ipv6 net.IP, err error)
}

// Implementations
resolver.NewSystem()                              // System DNS
resolver.NewUDP(addr, timeout)                    // UDP DNS
resolver.NewTCP(addr, timeout)                    // TCP DNS
resolver.NewTLS(addr, timeout, sni, insecure)     // DNS-over-TLS
resolver.NewHTTPS(addr, timeout, sni, insecure)   // DNS-over-HTTPS
```

### pkg/router

Combines ACL rules with outbounds for traffic routing:

```go
import "github.com/xflash-panda/acl-engine/pkg/router"

// Create router
r, _ := router.New(rules, outbounds, geoLoader,
    router.WithCacheSize(2048),
    router.WithResolver(resolver.NewUDP("8.8.8.8", 0)),
)

// Or from file
r, _ := router.NewFromFile("rules.acl", outbounds, geoLoader)

// Use router (implements outbound.Outbound interface)
conn, _ := r.DialTCP(&outbound.Addr{Host: "example.com", Port: 443})
udpConn, _ := r.DialUDP(&outbound.Addr{Host: "example.com", Port: 53})
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

### Supported Formats

| Format | GeoIP | GeoSite | Extension | Description |
|--------|-------|---------|-----------|-------------|
| DAT | Yes | Yes | `.dat` | V2Ray protobuf format |
| MMDB | Yes | No | `.mmdb` | MaxMind database format |
| MetaDB | Yes | No | `.metadb` | Clash Meta format |
| Sing | No | Yes | `.db` | sing-geosite binary format |

### GeoLoader Implementations

#### 1. AutoGeoLoader (Recommended)

```go
geoLoader := &acl.AutoGeoLoader{
    DataDir:        "./data",
    GeoIPFormat:    acl.GeoIPFormatMMDB,
    GeoIPURL:       acl.MetaCubeXGeoIPMMDBURL,
    GeoSiteFormat:  acl.GeoSiteFormatDAT,
    GeoSiteURL:     acl.MetaCubeXGeoSiteDatURL,
    UpdateInterval: 7 * 24 * time.Hour,
}
```

#### 2. FileGeoLoader

```go
geoLoader := acl.NewFileGeoLoader("./geoip.mmdb", "./geosite.dat")
```

#### 3. NilGeoLoader

```go
geoLoader := &acl.NilGeoLoader{}
```

### Available URL Constants

```go
acl.MetaCubeXGeoIPDatURL    // geoip.dat
acl.MetaCubeXGeoIPMMDBURL   // country.mmdb
acl.MetaCubeXGeoIPMetaDBURL // geoip.metadb
acl.MetaCubeXGeoSiteDatURL  // geosite.dat
acl.MetaCubeXGeoSiteDBURL   // geosite.db
```

## Integration with Other Frameworks

The `outbound.Outbound` interface is designed to be framework-agnostic. To integrate with other proxy frameworks (e.g., Hysteria, sing-box), create an adapter:

```go
// Example: Hysteria adapter
type HysteriaAdapter struct {
    outbound.Outbound
}

func (a *HysteriaAdapter) TCP(reqAddr string) (net.Conn, error) {
    host, port, _ := net.SplitHostPort(reqAddr)
    portInt, _ := strconv.Atoi(port)
    return a.Outbound.DialTCP(&outbound.Addr{
        Host: host,
        Port: uint16(portInt),
    })
}

func (a *HysteriaAdapter) UDP(reqAddr string) (server.UDPConn, error) {
    // Similar conversion...
}
```

## License

MIT License
