package cel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/pavillio/pav-edi/internal/domain"
)

// ValidationError indicates one or more CEL rules evaluated to false.
type ValidationError struct {
	RuleID  string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("rule %q failed", e.RuleID)
}

type compiledRule struct {
	rule domain.CELRule
	prog cel.Program
}

// RuleSet holds compiled CEL programs for repeated evaluation (e.g. config cache).
type RuleSet struct {
	rules []compiledRule
}

// NewRuleSet compiles rules once for reuse across many evaluations.
func NewRuleSet(rules []domain.CELRule) (*RuleSet, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for _, rule := range rules {
		ast, issues := claimEnv.Compile(rule.CEL)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("rule %q compile: %w", rule.ID, issues.Err())
		}
		prog, err := claimEnv.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("rule %q program: %w", rule.ID, err)
		}
		compiled = append(compiled, compiledRule{rule: rule, prog: prog})
	}
	return &RuleSet{rules: compiled}, nil
}

// Evaluate runs all compiled rules against bindings.
func (rs *RuleSet) Evaluate(bindings map[string]any) error {
	for _, cr := range rs.rules {
		ok, err := evalProgram(cr.rule, cr.prog, bindings)
		if err != nil {
			return fmt.Errorf("rule %q: %w", cr.rule.ID, err)
		}
		if !ok {
			msg := cr.rule.Message
			if msg == "" {
				msg = fmt.Sprintf("rule %q failed", cr.rule.ID)
			}
			return &ValidationError{RuleID: cr.rule.ID, Message: msg}
		}
	}
	return nil
}

// EvaluateAll compiles and evaluates rules (convenience for one-off checks).
func EvaluateAll(rules []domain.CELRule, bindings map[string]any) error {
	rs, err := NewRuleSet(rules)
	if err != nil {
		return err
	}
	return rs.Evaluate(bindings)
}

func evalProgram(rule domain.CELRule, prog cel.Program, bindings map[string]any) (bool, error) {
	out, _, err := prog.Eval(bindings)
	if err != nil {
		return false, fmt.Errorf("eval: %w", err)
	}
	if out.Type() != cel.BoolType {
		return false, fmt.Errorf("expression must return bool, got %v", out.Type())
	}
	return out.Value().(bool), nil
}
