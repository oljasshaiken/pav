package mapping

import (
	"fmt"
	"strings"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
)

// ResolvePath reads a dot-path from ClaimContext (patient.*, agency.*, claim.*,
// visit.* for the first visit, service_line.* for the first service line).
//
// claim.total_amount resolves from the first service line amount (Phase 1 single-line).
func ResolvePath(ctx domain.ClaimContext, path string) (string, error) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid path %q", path)
	}

	switch parts[0] {
	case "patient":
		return resolvePatient(ctx.Patient, parts[1])
	case "agency":
		return resolveAgency(ctx.Agency, parts[1])
	case "claim":
		return resolveClaim(ctx, parts[1])
	case "visit":
		return resolveVisit(firstVisit(ctx), parts[1])
	case "service_line":
		return resolveServiceLine(firstServiceLine(ctx), parts[1])
	default:
		return "", fmt.Errorf("unknown path prefix %q", path)
	}
}

func resolvePatient(p domain.Patient, field string) (string, error) {
	switch field {
	case "first_name":
		return p.FirstName, nil
	case "last_name":
		return p.LastName, nil
	case "medicaid_id":
		return p.MedicaidID, nil
	default:
		return "", fmt.Errorf("unknown patient field %q", field)
	}
}

func resolveAgency(a domain.Agency, field string) (string, error) {
	switch field {
	case "name":
		return a.Name, nil
	case "npi":
		return a.NPI, nil
	default:
		return "", fmt.Errorf("unknown agency field %q", field)
	}
}

func resolveClaim(ctx domain.ClaimContext, field string) (string, error) {
	switch field {
	case "claim_number":
		if ctx.Claim.ClaimNumber == nil {
			return "", fmt.Errorf("claim.claim_number is not set")
		}
		return *ctx.Claim.ClaimNumber, nil
	case "total_amount":
		sl, err := firstServiceLineOrErr(ctx)
		if err != nil {
			return "", err
		}
		if sl.Amount == nil {
			return "", fmt.Errorf("service_line.amount is not set")
		}
		return formatAmount(*sl.Amount), nil
	default:
		return "", fmt.Errorf("unknown claim field %q", field)
	}
}

func resolveVisit(v domain.Visit, field string) (string, error) {
	switch field {
	case "clock_in_time":
		if v.ClockInTime == nil {
			return "", fmt.Errorf("visit.clock_in_time is not set")
		}
		return formatCompactDateTime(*v.ClockInTime), nil
	default:
		return "", fmt.Errorf("unknown visit field %q", field)
	}
}

func resolveServiceLine(sl domain.ClaimServiceLine, field string) (string, error) {
	switch field {
	case "procedure_code":
		return sl.ProcedureCode, nil
	case "units":
		return formatUnits(sl.Units), nil
	case "amount":
		if sl.Amount == nil {
			return "", fmt.Errorf("service_line.amount is not set")
		}
		return formatAmount(*sl.Amount), nil
	case "diagnosis_codes":
		if len(sl.DiagnosisCodes) == 0 {
			return "", fmt.Errorf("service_line.diagnosis_codes is empty")
		}
		return sl.DiagnosisCodes[0], nil
	default:
		return "", fmt.Errorf("unknown service_line field %q", field)
	}
}

func firstVisit(ctx domain.ClaimContext) domain.Visit {
	if len(ctx.Visits) == 0 {
		return domain.Visit{}
	}
	return ctx.Visits[0]
}

func firstServiceLine(ctx domain.ClaimContext) domain.ClaimServiceLine {
	if len(ctx.ServiceLines) == 0 {
		return domain.ClaimServiceLine{}
	}
	return ctx.ServiceLines[0]
}

func firstServiceLineOrErr(ctx domain.ClaimContext) (domain.ClaimServiceLine, error) {
	if len(ctx.ServiceLines) == 0 {
		return domain.ClaimServiceLine{}, fmt.Errorf("claim has no service lines")
	}
	return ctx.ServiceLines[0], nil
}

func formatAmount(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func formatUnits(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%g", v)
}

func formatCompactDateTime(t time.Time) string {
	u := t.UTC()
	return u.Format("20060102") + "T" + u.Format("150405")
}
