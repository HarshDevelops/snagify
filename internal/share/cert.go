// Package share implements secure local-network baseline sharing: a temporary
// TLS server that serves a single sanitized baseline file, and a client that
// fetches it while pinning the server's certificate fingerprint.
//
// Scope (v0.2): same-LAN/VPN only. No NAT traversal, no cloud relay, no
// external servers. The server is short-lived (expiry + max-downloads) and
// never accepts uploads.
package share

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// ephemeralCert generates a self-signed TLS certificate valid for ttl, and
// returns the tls.Certificate plus its SHA-256 fingerprint (DER of the leaf).
func ephemeralCert(ttl time.Duration) (tls.Certificate, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, "", err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, "", err
	}

	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "snagify-share"},
		NotBefore:             now.Add(-1 * time.Minute),
		NotAfter:              now.Add(ttl + time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, "", err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
	return cert, Fingerprint(der), nil
}

// Fingerprint returns the SHA-256 of a certificate's DER bytes formatted as
// uppercase colon-separated hex (e.g. "3F:92:A1:..."), without a scheme prefix.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = strings.ToUpper(hex.EncodeToString([]byte{b}))
	}
	return strings.Join(parts, ":")
}

// NormalizePin strips an optional "sha256:" prefix and normalizes a pin to
// uppercase colon-separated hex for exact comparison.
func NormalizePin(pin string) string {
	pin = strings.TrimSpace(pin)
	pin = strings.TrimPrefix(pin, "sha256:")
	pin = strings.TrimPrefix(pin, "SHA256:")
	return strings.ToUpper(pin)
}

// PinString returns the canonical pin form including the scheme prefix.
func PinString(fingerprint string) string {
	return "sha256:" + fingerprint
}

// equalPins compares two pins in normalized form.
func equalPins(a, b string) bool {
	return NormalizePin(a) == NormalizePin(b)
}

// randomToken returns a URL-safe random token using crypto/rand.
func randomToken(nbytes int) (string, error) {
	buf := make([]byte, nbytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
