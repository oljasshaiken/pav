"use client";

import { useCallback, useEffect, useState } from "react";

import { StepTimeline } from "@/components/StepTimeline";
import {
  DEFAULT_CLAIM_ID,
  DEFAULT_GENERATED_AT,
  fetchBackends,
  pollSFNExecution,
  runWorkflow,
  startSFNRun,
  type BackendsStatus,
  type RunTrace,
  type StepRecord,
} from "@/lib/api";

type RunMode = "option1" | "option3" | "option3_sfn";

function statusBadge(ok: boolean | null, optional = false) {
  if (ok === null) return "bg-zinc-100 text-zinc-600";
  if (optional && !ok) return "bg-zinc-100 text-zinc-500";
  return ok ? "bg-emerald-100 text-emerald-800" : "bg-red-100 text-red-800";
}

export function ClaimRunner() {
  const [claimId, setClaimId] = useState(DEFAULT_CLAIM_ID);
  const [generatedAt, setGeneratedAt] = useState(DEFAULT_GENERATED_AT);
  const [mode, setMode] = useState<RunMode>("option1");
  const [skipPersist, setSkipPersist] = useState(true);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [trace, setTrace] = useState<RunTrace | null>(null);
  const [steps, setSteps] = useState<StepRecord[]>([]);
  const [backends, setBackends] = useState<BackendsStatus | null>(null);
  const [sfnStatus, setSfnStatus] = useState<string | null>(null);

  const refreshBackends = useCallback(async () => {
    try {
      setBackends(await fetchBackends());
    } catch {
      setBackends({
        postgres: "down",
        pipeline: "down",
        step_functions: "down",
      });
    }
  }, []);

  useEffect(() => {
    void refreshBackends();
  }, [refreshBackends]);

  async function handleRun() {
    setRunning(true);
    setError(null);
    setTrace(null);
    setSteps([]);
    setSfnStatus(null);

    try {
      if (mode === "option3_sfn") {
        const start = await startSFNRun(claimId);
        setSfnStatus(start.status);
        let done = false;
        while (!done) {
          const status = await pollSFNExecution(start.execution_arn);
          setSteps(status.steps);
          setSfnStatus(status.status);
          setTrace({
            claim_id: claimId,
            mode: "option3_sfn",
            success: status.success,
            steps: status.steps,
            failed_step: status.failed_step,
            result: status.result,
          });
          if (
            status.status === "SUCCEEDED" ||
            status.status === "FAILED" ||
            status.status === "TIMED_OUT" ||
            status.status === "ABORTED"
          ) {
            done = true;
          } else {
            await new Promise((r) => setTimeout(r, 1000));
          }
        }
      } else {
        const result = await runWorkflow(
          claimId,
          mode,
          generatedAt,
          skipPersist,
        );
        setTrace(result);
        setSteps(result.steps);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "run failed");
    } finally {
      setRunning(false);
    }
  }

  const sfnAvailable = backends?.step_functions === "ok";

  return (
    <div className="mx-auto max-w-3xl space-y-8 px-4 py-10">
      <header className="space-y-2">
        <h1 className="text-2xl font-semibold text-zinc-900">
          PAV Outbound Workflow
        </h1>
        <p className="text-sm text-zinc-600">
          Step-by-step view for Option 1 (rules pipeline) and Option 3
          (workflow / Step Functions).
        </p>
      </header>

      {backends && (
        <div className="flex flex-wrap gap-2 text-xs">
          {(
            [
              ["Postgres", backends.postgres === "ok", false],
              ["Pipeline (in-process)", backends.pipeline === "ok", false],
              ["Step Functions", backends.step_functions === "ok", true],
            ] as const
          ).map(([label, ok, optional]) => (
            <span
              key={label}
              className={`rounded-full px-3 py-1 ${statusBadge(ok, optional)}`}
            >
              {label}: {ok ? "ok" : optional ? "off" : "down"}
            </span>
          ))}
          {backends.rules_engine_http === "down" && (
            <span
              className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-500"
              title="Only required for make compare — run make run-rules in another terminal"
            >
              Rules HTTP :8081: off (optional)
            </span>
          )}
          {backends.rules_engine_http === "ok" && (
            <span className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-600">
              Rules HTTP :8081: ok
            </span>
          )}
        </div>
      )}

      <section className="space-y-4 rounded-xl border border-zinc-200 bg-white p-6 shadow-sm">
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm">
            <span className="font-medium text-zinc-700">Claim ID</span>
            <input
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm"
              value={claimId}
              onChange={(e) => setClaimId(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium text-zinc-700">Generated at (RFC3339)</span>
            <input
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm"
              value={generatedAt}
              onChange={(e) => setGeneratedAt(e.target.value)}
              disabled={mode === "option3_sfn"}
            />
          </label>
        </div>

        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-zinc-700">Mode</legend>
          <div className="flex flex-wrap gap-4 text-sm">
            {(
              [
                ["option1", "Option 1 — Rules pipeline"],
                ["option3", "Option 3 — Workflow (local)"],
                ["option3_sfn", "Option 3 — Step Functions"],
              ] as const
            ).map(([value, label]) => (
              <label key={value} className="flex items-center gap-2">
                <input
                  type="radio"
                  name="mode"
                  value={value}
                  checked={mode === value}
                  onChange={() => setMode(value)}
                  disabled={value === "option3_sfn" && !sfnAvailable}
                />
                <span
                  className={
                    value === "option3_sfn" && !sfnAvailable
                      ? "text-zinc-400"
                      : ""
                  }
                  title={
                    value === "option3_sfn" && !sfnAvailable
                      ? "Requires LocalStack + make sam-deploy-localstack"
                      : undefined
                  }
                >
                  {label}
                </span>
              </label>
            ))}
          </div>
        </fieldset>

        {mode !== "option3_sfn" && (
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={skipPersist}
              onChange={(e) => setSkipPersist(e.target.checked)}
            />
            Skip persist (dry-run, no DB/S3 write)
          </label>
        )}

        <button
          type="button"
          onClick={() => void handleRun()}
          disabled={running || (mode === "option3_sfn" && !sfnAvailable)}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {running ? "Running…" : "Run workflow"}
        </button>

        {sfnStatus && (
          <p className="text-sm text-zinc-600">SFN status: {sfnStatus}</p>
        )}
        {error && <p className="text-sm text-red-700">{error}</p>}
      </section>

      {(steps.length > 0 || trace) && (
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-medium text-zinc-900">Steps</h2>
            {trace && (
              <span
                className={`rounded-full px-3 py-1 text-xs font-medium ${
                  trace.success
                    ? "bg-emerald-100 text-emerald-800"
                    : "bg-red-100 text-red-800"
                }`}
              >
                {trace.success ? "Success" : "Failed"}
              </span>
            )}
          </div>
          <StepTimeline steps={steps} failedStep={trace?.failed_step} />
        </section>
      )}

      {trace?.result?.edi && (
        <section className="space-y-2">
          <h2 className="text-lg font-medium text-zinc-900">EDI preview</h2>
          <pre className="overflow-x-auto rounded-lg border border-zinc-200 bg-zinc-50 p-4 text-xs text-zinc-800">
            {trace.result.edi.slice(0, 500)}
            {trace.result.edi.length > 500 ? "…" : ""}
          </pre>
        </section>
      )}
    </div>
  );
}
