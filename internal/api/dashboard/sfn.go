package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/pavillio/pav-edi/internal/pipeline"
)

const outboundStateMachineName = "pav-edi-outbound-claim"

var sfnStateToStep = map[string]string{
	"LoadClaim": pipeline.StepLoad,
	"RulesPre":  pipeline.StepRulesPre,
	"Transform": pipeline.StepTransform,
	"RulesPost": pipeline.StepRulesPost,
	"Persist":   pipeline.StepPersist,
}

// SFNClient wraps LocalStack/AWS Step Functions for dashboard polling.
type SFNClient struct {
	client *sfn.Client
}

// SFNStartResponse is returned when an execution is started.
type SFNStartResponse struct {
	ExecutionARN string `json:"execution_arn"`
	Status       string `json:"status"`
	ClaimID      string `json:"claim_id"`
	Mode         string `json:"mode"`
}

// SFNExecutionStatus is the polled execution view for the dashboard.
type SFNExecutionStatus struct {
	ExecutionARN string                   `json:"execution_arn"`
	Status       string                   `json:"status"`
	Success      bool                     `json:"success"`
	Steps        []pipeline.StepRecord    `json:"steps"`
	FailedStep   string                   `json:"failed_step,omitempty"`
	Result       *pipeline.GenerateResult `json:"result,omitempty"`
}

// NewSFNClient builds a Step Functions client (LocalStack when endpoint is set).
func NewSFNClient(ctx context.Context, endpoint, region string) (*SFNClient, error) {
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		return nil, err
	}
	client := sfn.NewFromConfig(cfg, func(o *sfn.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &SFNClient{client: client}, nil
}

func (c *SFNClient) Available(ctx context.Context) bool {
	out, err := c.client.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
	if err != nil {
		return false
	}
	for _, sm := range out.StateMachines {
		if aws.ToString(sm.Name) == outboundStateMachineName {
			return true
		}
	}
	return false
}

func (c *SFNClient) stateMachineARN(ctx context.Context) (string, error) {
	out, err := c.client.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
	if err != nil {
		return "", err
	}
	for _, sm := range out.StateMachines {
		if aws.ToString(sm.Name) == outboundStateMachineName {
			return aws.ToString(sm.StateMachineArn), nil
		}
	}
	return "", fmt.Errorf("state machine %q not found", outboundStateMachineName)
}

// StartExecution begins OutboundClaimWorkflow for a claim.
func (c *SFNClient) StartExecution(ctx context.Context, claimID string) (SFNStartResponse, error) {
	arn, err := c.stateMachineARN(ctx)
	if err != nil {
		return SFNStartResponse{}, err
	}
	input, _ := json.Marshal(map[string]string{"claim_id": claimID})
	out, err := c.client.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: aws.String(arn),
		Input:           aws.String(string(input)),
	})
	if err != nil {
		return SFNStartResponse{}, err
	}
	return SFNStartResponse{
		ExecutionARN: aws.ToString(out.ExecutionArn),
		Status:       "RUNNING",
		ClaimID:      claimID,
		Mode:         "option3_sfn",
	}, nil
}

// ExecutionStatus polls execution history and maps SFN states to unified steps.
func (c *SFNClient) ExecutionStatus(ctx context.Context, executionARN string) (SFNExecutionStatus, error) {
	desc, err := c.client.DescribeExecution(ctx, &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(executionARN),
	})
	if err != nil {
		return SFNExecutionStatus{}, err
	}

	rec := pipeline.NewMemoryRecorder()
	status := string(desc.Status)

	history, err := c.client.GetExecutionHistory(ctx, &sfn.GetExecutionHistoryInput{
		ExecutionArn: aws.String(executionARN),
	})
	if err != nil {
		return SFNExecutionStatus{}, err
	}

	activeStep := ""
	for _, event := range history.Events {
		switch event.Type {
		case sfntypes.HistoryEventTypeTaskStateEntered:
			if event.StateEnteredEventDetails == nil {
				continue
			}
			stepID := sfnStateToStep[aws.ToString(event.StateEnteredEventDetails.Name)]
			if stepID == "" {
				continue
			}
			rec.Begin(stepID)
			activeStep = stepID
		case sfntypes.HistoryEventTypeTaskStateExited:
			if event.StateExitedEventDetails == nil {
				continue
			}
			stepID := sfnStateToStep[aws.ToString(event.StateExitedEventDetails.Name)]
			if stepID == "" {
				continue
			}
			rec.End(stepID, nil)
			if activeStep == stepID {
				activeStep = ""
			}
		case sfntypes.HistoryEventTypeTaskFailed:
			errMsg := "task failed"
			if event.TaskFailedEventDetails != nil {
				if cause := aws.ToString(event.TaskFailedEventDetails.Cause); cause != "" {
					errMsg = cause
				} else if e := aws.ToString(event.TaskFailedEventDetails.Error); e != "" {
					errMsg = e
				}
			}
			if activeStep != "" {
				rec.End(activeStep, fmt.Errorf("%s", errMsg))
				activeStep = ""
			}
		}
	}

	resp := SFNExecutionStatus{
		ExecutionARN: executionARN,
		Status:       status,
		Success:      status == string(sfntypes.ExecutionStatusSucceeded),
		Steps:        rec.Snapshot(),
		FailedStep:   rec.FailedStep(),
	}

	if desc.Output != nil && aws.ToString(desc.Output) != "" {
		var payload struct {
			ClaimID string `json:"claim_id"`
			Persist struct {
				S3Key string `json:"s3_key"`
			} `json:"persist"`
			Transform struct {
				Document struct {
					Raw           string    `json:"raw"`
					GeneratedAt   time.Time `json:"generated_at"`
					ConfigVersion int32     `json:"config_version"`
				} `json:"document"`
			} `json:"transform"`
		}
		if err := json.Unmarshal([]byte(aws.ToString(desc.Output)), &payload); err == nil && payload.Transform.Document.Raw != "" {
			resp.Result = &pipeline.GenerateResult{
				ClaimID:       payload.ClaimID,
				ConfigVersion: payload.Transform.Document.ConfigVersion,
				EDI:           payload.Transform.Document.Raw,
				S3Key:         payload.Persist.S3Key,
				GeneratedAt:   payload.Transform.Document.GeneratedAt,
			}
		}
	}

	return resp, nil
}
