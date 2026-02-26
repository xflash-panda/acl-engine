package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/xflash-panda/acl-engine/pkg/acl"
	"github.com/xflash-panda/acl-engine/pkg/outbound"
	"github.com/xflash-panda/acl-engine/pkg/router"

	"gopkg.in/yaml.v3"
)

// BuildOptions provides additional options for building a Router from config.
// These are typically set programmatically rather than from the config file.
type BuildOptions struct {
	GeoLoader acl.GeoLoader
	CacheSize int // LRU cache size for rule matching (default: 1024)
}

// Config is the top-level configuration structure.
type Config struct {
	Outbounds []OutboundConfig `yaml:"outbounds"`
	ACL       ACLConfig        `yaml:"acl"`
}

// OutboundConfig defines a named outbound.
type OutboundConfig struct {
	Name   string        `yaml:"name"`
	Type   string        `yaml:"type"` // direct, socks5, http, reject
	Direct *DirectConfig `yaml:"direct"`
	SOCKS5 *SOCKS5Config `yaml:"socks5"`
	HTTP   *HTTPConfig   `yaml:"http"`
}

// DirectConfig configures a direct outbound.
type DirectConfig struct {
	Mode       string `yaml:"mode"`       // auto, 64, 46, 6, 4
	BindIPv4   string `yaml:"bindIPv4"`   // IPv4 address to bind
	BindIPv6   string `yaml:"bindIPv6"`   // IPv6 address to bind
	BindDevice string `yaml:"bindDevice"` // network device name (Linux only)
	FastOpen   bool   `yaml:"fastOpen"`   // TCP Fast Open
}

// SOCKS5Config configures a SOCKS5 outbound.
type SOCKS5Config struct {
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// HTTPConfig configures an HTTP/HTTPS proxy outbound.
type HTTPConfig struct {
	URL      string `yaml:"url"` // http://[user:pass@]host:port or https://...
	Insecure bool   `yaml:"insecure"`
}

// ACLConfig configures ACL rules.
type ACLConfig struct {
	File   string   `yaml:"file"`   // load rules from file
	Inline []string `yaml:"inline"` // inline rules
}

// Load reads a YAML config file and returns a configured Router.
// GeoLoader is provided by the caller via BuildOptions since geo data
// configuration (paths, update intervals, URLs) is managed programmatically.
func Load(filename string, bopts *BuildOptions) (*router.Router, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return Parse(data, bopts)
}

// Parse parses YAML data and returns a configured Router.
func Parse(data []byte, bopts *BuildOptions) (*router.Router, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return Build(&cfg, bopts)
}

// Build constructs a Router from a parsed Config.
func Build(cfg *Config, bopts *BuildOptions) (*router.Router, error) {
	entries, err := buildOutbounds(cfg.Outbounds)
	if err != nil {
		return nil, fmt.Errorf("build outbounds: %w", err)
	}

	rules, err := buildRules(&cfg.ACL)
	if err != nil {
		return nil, fmt.Errorf("build rules: %w", err)
	}

	var geoLoader acl.GeoLoader = &acl.NilGeoLoader{}
	if bopts != nil && bopts.GeoLoader != nil {
		geoLoader = bopts.GeoLoader
	}

	var opts []router.Option
	if bopts != nil && bopts.CacheSize > 0 {
		opts = append(opts, router.WithCacheSize(bopts.CacheSize))
	}

	return router.New(rules, entries, geoLoader, opts...)
}

func buildOutbounds(configs []OutboundConfig) ([]router.OutboundEntry, error) {
	entries := make([]router.OutboundEntry, 0, len(configs))
	for i, cfg := range configs {
		if cfg.Name == "" {
			return nil, fmt.Errorf("outbound[%d]: name is required", i)
		}
		ob, err := buildOutbound(&cfg)
		if err != nil {
			return nil, fmt.Errorf("outbound[%d] %q: %w", i, cfg.Name, err)
		}
		entries = append(entries, router.OutboundEntry{
			Name:     cfg.Name,
			Outbound: ob,
		})
	}
	return entries, nil
}

func buildOutbound(cfg *OutboundConfig) (outbound.Outbound, error) {
	switch strings.ToLower(cfg.Type) {
	case "direct":
		return buildDirect(cfg.Direct)
	case "socks5":
		return buildSOCKS5(cfg.SOCKS5)
	case "http":
		return buildHTTP(cfg.HTTP)
	case "reject":
		return outbound.NewReject(), nil
	default:
		return nil, fmt.Errorf("unknown outbound type: %q", cfg.Type)
	}
}

func buildDirect(cfg *DirectConfig) (outbound.Outbound, error) {
	opts := outbound.DirectOptions{}
	if cfg != nil {
		mode, err := parseDirectMode(cfg.Mode)
		if err != nil {
			return nil, err
		}
		opts.Mode = mode
		if cfg.BindDevice != "" && (cfg.BindIPv4 != "" || cfg.BindIPv6 != "") {
			return nil, fmt.Errorf("bindDevice is mutually exclusive with bindIPv4/bindIPv6")
		}
		if cfg.BindIPv4 != "" {
			ip := net.ParseIP(cfg.BindIPv4)
			if ip == nil {
				return nil, fmt.Errorf("invalid bindIPv4: %q", cfg.BindIPv4)
			}
			opts.BindIP4 = ip
		}
		if cfg.BindIPv6 != "" {
			ip := net.ParseIP(cfg.BindIPv6)
			if ip == nil {
				return nil, fmt.Errorf("invalid bindIPv6: %q", cfg.BindIPv6)
			}
			opts.BindIP6 = ip
		}
		opts.DeviceName = cfg.BindDevice
		opts.FastOpen = cfg.FastOpen
	}
	return outbound.NewDirectWithOptions(opts)
}

func parseDirectMode(s string) (outbound.DirectMode, error) {
	switch strings.ToLower(s) {
	case "", "auto":
		return outbound.DirectModeAuto, nil
	case "64":
		return outbound.DirectMode64, nil
	case "46":
		return outbound.DirectMode46, nil
	case "6":
		return outbound.DirectMode6, nil
	case "4":
		return outbound.DirectMode4, nil
	default:
		return 0, fmt.Errorf("unknown direct mode: %q (valid: auto, 64, 46, 6, 4)", s)
	}
}

func buildSOCKS5(cfg *SOCKS5Config) (outbound.Outbound, error) {
	if cfg == nil {
		return nil, fmt.Errorf("socks5 config is required for type socks5")
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("socks5 addr is required")
	}
	return outbound.NewSOCKS5(cfg.Addr, cfg.Username, cfg.Password), nil
}

func buildHTTP(cfg *HTTPConfig) (outbound.Outbound, error) {
	if cfg == nil {
		return nil, fmt.Errorf("http config is required for type http")
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("http url is required")
	}
	return outbound.NewHTTP(cfg.URL, cfg.Insecure)
}

func buildRules(cfg *ACLConfig) (string, error) {
	if cfg.File != "" && len(cfg.Inline) > 0 {
		return "", fmt.Errorf("cannot specify both acl.file and acl.inline")
	}
	if cfg.File != "" {
		data, err := os.ReadFile(cfg.File)
		if err != nil {
			return "", fmt.Errorf("read acl file: %w", err)
		}
		return string(data), nil
	}
	return strings.Join(cfg.Inline, "\n"), nil
}
