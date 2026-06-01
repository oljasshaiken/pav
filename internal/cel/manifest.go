package cel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/pkg/x12"
)

var manifestEnv *cel.Env

func init() {
	var err error
	manifestEnv, err = cel.NewEnv(
		cel.Declarations(
			decls.NewVar("manifest", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("cel manifest env: %v", err))
	}
}

// ManifestBindings exposes normalized X12 manifest fields to CEL business rules.
func ManifestBindings(m x12.Manifest) map[string]any {
	segments := make([]map[string]any, len(m.Segments))
	tags := make([]string, len(m.Segments))
	for i, seg := range m.Segments {
		segments[i] = map[string]any{
			"loop":     seg.Loop,
			"segment":  seg.Segment,
			"elements": seg.Elements,
		}
		tags[i] = seg.Segment
	}
	return map[string]any{
		"manifest": map[string]any{
			"segments":     segments,
			"segment_tags": tags,
		},
	}
}

// EvaluateManifestAll evaluates rules against manifest bindings.
func EvaluateManifestAll(rules []domain.CELRule, bindings map[string]any) error {
	for _, rule := range rules {
		ast, issues := manifestEnv.Compile(rule.CEL)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("rule %q compile: %w", rule.ID, issues.Err())
		}
		prog, err := manifestEnv.Program(ast)
		if err != nil {
			return fmt.Errorf("rule %q program: %w", rule.ID, err)
		}
		out, _, err := prog.Eval(bindings)
		if err != nil {
			return fmt.Errorf("rule %q eval: %w", rule.ID, err)
		}
		if out.Type() != cel.BoolType {
			return fmt.Errorf("rule %q must return bool", rule.ID)
		}
		if !out.Value().(bool) {
			msg := rule.Message
			if msg == "" {
				msg = fmt.Sprintf("rule %q failed", rule.ID)
			}
			return &ValidationError{RuleID: rule.ID, Message: msg}
		}
	}
	return nil
}
