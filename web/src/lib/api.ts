export type StepStatus = "pending" | "running" | "ok" | "failed" | "skipped";

export type StepRecord = {
  id: string;
  status: StepStatus;
  duration_ms?: number;
  error?: string;
};

export type GenerateResult = {
  claim_id: string;
  config_version: number;
  edi: string;
  s3_key?: string;
  generated_at: string;
};

export type RunTrace = {
  claim_id: string;
  mode: string;
  success: boolean;
  steps: StepRecord[];
  failed_step?: string;
  result?: GenerateResult;
};

export type BackendsStatus = {
  postgres: string;
  pipeline: string;
  rules_engine_http?: string;
  step_functions: string;
};

export type SFNStartResponse = {
  execution_arn: string;
  status: string;
  claim_id: string;
  mode: string;
};

export type SFNExecutionStatus = {
  execution_arn: string;
  status: string;
  success: boolean;
  steps: StepRecord[];
  failed_step?: string;
  result?: GenerateResult;
};

const API_BASE =
  process.env.NEXT_PUBLIC_DASHBOARD_API_URL ?? "http://localhost:8083";

export async function fetchBackends(): Promise<BackendsStatus> {
  const res = await fetch(`${API_BASE}/api/backends`, { cache: "no-store" });
  if (!res.ok) throw new Error("backends probe failed");
  return res.json();
}

export async function runWorkflow(
  claimId: string,
  mode: "option1" | "option3",
  generatedAt: string,
  skipPersist = true,
): Promise<RunTrace> {
  const params = new URLSearchParams({
    mode,
    skip_persist: String(skipPersist),
    generated_at: generatedAt,
  });
  const res = await fetch(
    `${API_BASE}/api/claims/${claimId}/run?${params.toString()}`,
    { method: "POST", cache: "no-store" },
  );
  const body = await res.json();
  if (body.error && !body.steps) {
    throw new Error(body.error.message ?? "run failed");
  }
  return body as RunTrace;
}

export async function startSFNRun(claimId: string): Promise<SFNStartResponse> {
  const res = await fetch(`${API_BASE}/api/claims/${claimId}/run-sfn`, {
    method: "POST",
    cache: "no-store",
  });
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.error?.message ?? "sfn start failed");
  }
  return body as SFNStartResponse;
}

export async function pollSFNExecution(
  executionArn: string,
): Promise<SFNExecutionStatus> {
  const params = new URLSearchParams({ arn: executionArn });
  const res = await fetch(
    `${API_BASE}/api/executions?${params.toString()}`,
    { cache: "no-store" },
  );
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.error?.message ?? "sfn poll failed");
  }
  return body as SFNExecutionStatus;
}

export const STEP_LABELS: Record<string, string> = {
  load: "Load claim + payer config",
  rules_pre: "Pre-transform rules (CEL + EVV)",
  transform: "Generate 837P",
  rules_post: "Post-transform rules",
  persist: "Persist EDI (Postgres + S3)",
};

export const DEFAULT_CLAIM_ID = "00000000-0000-4000-8000-000000000001";
export const DEFAULT_GENERATED_AT = "2026-05-31T12:00:00Z";
