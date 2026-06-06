package capture

import (
	"fmt"
	"net"
	"time"

	"github.com/HarshDevelops/snagify/internal/model"
)

// commonPorts maps frequently used dev ports to a human service label.
var commonPorts = map[int]string{
	3000:  "dev server",
	3306:  "MySQL",
	5000:  "dev server",
	5173:  "Vite",
	5432:  "Postgres",
	6379:  "Redis",
	8000:  "dev server",
	8080:  "HTTP alt",
	8443:  "HTTPS alt",
	9000:  "service",
	9200:  "Elasticsearch",
	27017: "MongoDB",
}

// capturePorts checks which common dev ports are currently listening on
// localhost. It uses a short TCP dial which is fully cross-platform and needs
// no elevated privileges.
//
// Limitation (v0.1): this only reports whether something is accepting TCP
// connections on 127.0.0.1. It does not identify the owning process and does
// not see services bound only to other interfaces. This is an intentional MVP
// tradeoff in favor of portability over the ss/lsof/netstat approach.
func capturePorts() model.Services {
	ports := make(map[int]model.PortInfo, len(commonPorts))
	for port, service := range commonPorts {
		ports[port] = model.PortInfo{
			Port:      port,
			Service:   service,
			Listening: isListening(port),
		}
	}
	return model.Services{Ports: ports}
}

// isListening reports whether something accepts TCP connections on the port.
func isListening(port int) bool {
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
