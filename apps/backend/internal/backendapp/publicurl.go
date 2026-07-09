package backendapp

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/AvatarGanymede/pcraft/internal/common/config"
)

// envPublicBaseURL is the explicit override for the externally reachable
// pcraft origin used in outbound notification deep-links.
const envPublicBaseURL = "PCRAFT_PUBLIC_BASE_URL"

// resolvePublicBaseURL determines the externally reachable base URL used to
// build task deep-links in outbound notifications (e.g. the Lark bot).
//
// Precedence:
//  1. PCRAFT_PUBLIC_BASE_URL (explicit override; trailing slash trimmed).
//  2. http://<outbound-ip>:<server-port>, where the outbound IP is the local
//     address this host would use to reach the network — never "localhost".
//
// Returns "" when no override is set and no usable IP can be determined; the
// Lark provider then simply omits the link.
func resolvePublicBaseURL(cfg *config.Config) string {
	if override := strings.TrimSpace(os.Getenv(envPublicBaseURL)); override != "" {
		return strings.TrimRight(override, "/")
	}
	ip := outboundIP()
	if ip == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", ip, cfg.Server.Port)
}

// outboundIP returns the preferred outbound IPv4 address of this host. It opens
// a throwaway UDP "connection" to a public address (no packets are actually
// sent) and reads back the local address the OS routing table selected. Falls
// back to scanning interface addresses. Returns "" when nothing usable exists.
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer func() { _ = conn.Close() }()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil && !addr.IP.IsLoopback() {
			if ip4 := addr.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return firstNonLoopbackIPv4()
}

// firstNonLoopbackIPv4 scans local interface addresses for the first non-
// loopback IPv4 address.
func firstNonLoopbackIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}
