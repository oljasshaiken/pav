package cel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/pavillio/pav-edi/internal/domain"
)

var claimEnv *cel.Env

func init() {
	var err error
	claimEnv, err = cel.NewEnv(
		cel.Declarations(
			decls.NewVar("authorization", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("service_line", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("visit", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("patient", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("agency", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("claim", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("cel claim env: %v", err))
	}
}

// ClaimEnv returns the shared CEL environment for claim validation.
func ClaimEnv() *cel.Env {
	return claimEnv
}

// ClaimBindings maps domain.ClaimContext to CEL variable bindings.
func ClaimBindings(ctx domain.ClaimContext) map[string]any {
	sl := domain.ClaimServiceLine{}
	if len(ctx.ServiceLines) > 0 {
		sl = ctx.ServiceLines[0]
	}
	visit := domain.Visit{}
	if len(ctx.Visits) > 0 {
		visit = ctx.Visits[0]
	}
	return map[string]any{
		"authorization": map[string]any{
			"service_type": ctx.Authorization.ServiceType,
		},
		"service_line": map[string]any{
			"diagnosis_codes": sl.DiagnosisCodes,
			"procedure_code":  sl.ProcedureCode,
			"units":           sl.Units,
		},
		"visit": visitBindings(visit),
		"patient": map[string]any{
			"medicaid_id": ctx.Patient.MedicaidID,
			"first_name":  ctx.Patient.FirstName,
			"last_name":   ctx.Patient.LastName,
		},
		"agency": map[string]any{
			"name":  ctx.Agency.Name,
			"state": ctx.Agency.State,
		},
		"claim": map[string]any{
			"claim_number": ctx.Claim.ClaimNumber,
			"state":        ctx.Claim.State,
			"payer_id":     ctx.Claim.PayerID,
		},
	}
}
