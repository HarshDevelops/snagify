package model

// DBProbeResult records an unauthenticated protocol-level probe of a database
// service. No credentials are sent. The probe confirms that the server speaks
// the expected wire protocol on the target port, and where possible reports
// a non-secret ServerVersion string. Tests must not assume ServerVersion is
// populated; many servers require authentication to disclose it.
type DBProbeResult struct {
	Name          string `json:"name"`
	Backend       string `json:"backend,omitempty"` // "postgres" | "mysql" | "redis"
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Required      bool   `json:"required"`
	Reachable     bool   `json:"reachable"`
	HandshakeOK   bool   `json:"handshake_ok"`
	ServerVersion string `json:"server_version,omitempty"`
	Error         string `json:"error,omitempty"`
}
