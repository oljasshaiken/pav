package cel_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/domain"
)

func TestBenchmarkGate_50RulesUnder1msP99(t *testing.T) {
	rules := make([]domain.CELRule, 50)
	for i := range rules {
		rules[i] = domain.CELRule{
			ID:      fmt.Sprintf("rule_%d", i),
			CEL:     `authorization.service_type != "home_health" || size(service_line.diagnosis_codes) > 0`,
			Message: "diagnosis required",
		}
	}
	bindings := cel.ClaimBindings(domain.ClaimContext{
		Authorization: domain.Authorization{ServiceType: "home_health"},
		ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: []string{"Z9999"}}},
	})

	rs, err := cel.NewRuleSet(rules)
	if err != nil {
		t.Fatal(err)
	}

	const iterations = 1000
	latencies := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		if err := rs.Evaluate(bindings); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		latencies[i] = time.Since(start)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p99 := latencies[len(latencies)*99/100]
	if p99 >= time.Millisecond {
		t.Fatalf("p99 latency %v exceeds 1ms gate", p99)
	}
}
