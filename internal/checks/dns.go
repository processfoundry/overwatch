package checks

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type DNSChecker struct{}

func (d *DNSChecker) Check(ctx context.Context, check spec.CheckSpec) spec.CheckResult {
	start := time.Now()
	result := spec.CheckResult{
		CheckName: check.Name,
		Timestamp: start,
	}

	var resolver net.Resolver
	recordType := strings.ToUpper(check.RecordType)
	if recordType == "" {
		recordType = "A"
	}

	switch recordType {
	case "A", "AAAA":
		addrs, err := resolver.LookupHost(ctx, check.Target)
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = spec.StatusDown
			result.Error = err.Error()
			return result
		}
		if len(addrs) == 0 {
			result.Status = spec.StatusDown
			result.Error = "no addresses returned"
			return result
		}
		result.Status = spec.StatusUp
		result.Detail = map[string]any{
			"recordType": recordType,
			"records":    addrs,
		}

	case "MX":
		mxs, err := resolver.LookupMX(ctx, check.Target)
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = spec.StatusDown
			result.Error = err.Error()
			return result
		}
		if len(mxs) == 0 {
			result.Status = spec.StatusDown
			result.Error = "no MX records returned"
			return result
		}
		records := make([]map[string]any, len(mxs))
		for i, mx := range mxs {
			records[i] = map[string]any{
				"host":     strings.TrimSuffix(mx.Host, "."),
				"priority": mx.Pref,
			}
		}
		result.Status = spec.StatusUp
		result.Detail = map[string]any{
			"recordType": "MX",
			"records":    records,
		}

	case "NS":
		nss, err := resolver.LookupNS(ctx, check.Target)
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = spec.StatusDown
			result.Error = err.Error()
			return result
		}
		if len(nss) == 0 {
			result.Status = spec.StatusDown
			result.Error = "no NS records returned"
			return result
		}
		records := make([]string, len(nss))
		for i, ns := range nss {
			records[i] = strings.TrimSuffix(ns.Host, ".")
		}
		result.Status = spec.StatusUp
		result.Detail = map[string]any{
			"recordType": "NS",
			"records":    records,
		}

	case "TXT":
		txts, err := resolver.LookupTXT(ctx, check.Target)
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = spec.StatusDown
			result.Error = err.Error()
			return result
		}
		if len(txts) == 0 {
			result.Status = spec.StatusDown
			result.Error = "no TXT records returned"
			return result
		}
		result.Status = spec.StatusUp
		result.Detail = map[string]any{
			"recordType": "TXT",
			"records":    txts,
		}

	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, check.Target)
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = spec.StatusDown
			result.Error = err.Error()
			return result
		}
		result.Status = spec.StatusUp
		result.Detail = map[string]any{
			"recordType": "CNAME",
			"records":    []string{strings.TrimSuffix(cname, ".")},
		}

	default:
		result.Duration = time.Since(start)
		result.Status = spec.StatusDown
		result.Error = fmt.Sprintf("unsupported record type: %s", recordType)
	}

	return result
}
