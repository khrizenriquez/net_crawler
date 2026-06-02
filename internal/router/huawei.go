package router

import (
	"errors"
	"net"
	"net/url"

	"github.com/duku/net-lab/internal/model"
)

var ErrUnsupportedFirmware = errors.New("Huawei firmware does not expose a configured read-only WAN counter endpoint")

type HuaweiConfig struct {
	BaseURL  string
	Username string
	Password string
}

type HuaweiCollector struct{ config HuaweiConfig }

func NewHuaweiCollector(config HuaweiConfig) (*HuaweiCollector, error) {
	parsed, err := url.Parse(config.BaseURL)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return nil, errors.New("Huawei base URL must be a valid local HTTP URL")
	}
	if !isPrivateHost(parsed.Hostname()) {
		return nil, errors.New("Huawei collector only allows private local addresses")
	}
	return &HuaweiCollector{config: config}, nil
}

// Poll intentionally reports unsupported until a firmware-specific read-only
// counter endpoint is confirmed. It must never guess paths or mutate router state.
func (c *HuaweiCollector) Poll() (model.RouterSample, error) {
	return model.RouterSample{Status: "unsupported", Message: ErrUnsupportedFirmware.Error()}, ErrUnsupportedFirmware
}

func isPrivateHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsPrivate()
}
