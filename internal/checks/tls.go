package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

const certExpiryWarning = 7 * 24 * time.Hour // 7 days

func humanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

type TLSChecker struct{}

func (t *TLSChecker) Check(ctx context.Context, check spec.CheckSpec) spec.CheckResult {
	start := time.Now()
	result := spec.CheckResult{
		CheckName: check.Name,
		Timestamp: start,
	}

	host, _, err := net.SplitHostPort(check.Target)
	if err != nil {
		host = check.Target
		check.Target = check.Target + ":443"
	}

	d := tls.Dialer{Config: &tls.Config{ServerName: host}}
	conn, err := d.DialContext(ctx, "tcp", check.Target)
	result.Duration = time.Since(start)
	if err != nil {
		result.Status = spec.StatusDown
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		result.Status = spec.StatusDown
		result.Error = "no certificates presented"
		return result
	}

	leaf := certs[0]
	until := time.Until(leaf.NotAfter)

	result.Detail = map[string]any{
		"subject":       leaf.Subject.CommonName,
		"issuer":        leaf.Issuer.CommonName,
		"expiresAt":     leaf.NotAfter.Format("2006-01-02"),
		"daysRemaining": int(until.Hours() / 24),
	}

	warnDuration := certExpiryWarning
	if check.WarnDays > 0 {
		warnDuration = time.Duration(check.WarnDays) * 24 * time.Hour
	}

	if until <= 0 {
		result.Status = spec.StatusDown
		result.Error = fmt.Sprintf("certificate expired %s ago", humanDuration(-until))
	} else if until < warnDuration {
		result.Status = spec.StatusDegraded
		result.Error = fmt.Sprintf("certificate expires in %s", humanDuration(until))
	} else {
		result.Status = spec.StatusUp
	}

	return result
}
