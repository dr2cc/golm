package urlutil

import (
	"net"
	"strings"
)

// JoinHostPort очищает хост от схем http/https и объединяет с портом.
func JoinHostPort(host, port string) string {
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if port == "" {
		port = "80"
	}
	return net.JoinHostPort(host, port)
}
