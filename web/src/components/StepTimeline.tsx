"use client";

import { STEP_LABELS, type StepRecord, type StepStatus } from "@/lib/api";

const STATUS_STYLES: Record<
  StepStatus,
  { dot: string; label: string; ring: string }
> = {
  pending: {
    dot: "bg-zinc-300",
    label: "Pending",
    ring: "border-zinc-200",
  },
  running: {
    dot: "bg-blue-500 animate-pulse",
    label: "Running",
    ring: "border-blue-300",
  },
  ok: {
    dot: "bg-emerald-500",
    label: "OK",
    ring: "border-emerald-200",
  },
  failed: {
    dot: "bg-red-500",
    label: "Failed",
    ring: "border-red-300",
  },
  skipped: {
    dot: "bg-zinc-400",
    label: "Skipped",
    ring: "border-zinc-200",
  },
};

type Props = {
  steps: StepRecord[];
  failedStep?: string;
};

export function StepTimeline({ steps, failedStep }: Props) {
  const ordered: StepRecord[] =
    steps.length > 0
      ? steps
      : Object.keys(STEP_LABELS).map((id) => ({
          id,
          status: "pending" as StepStatus,
        }));

  return (
    <ol className="space-y-3" aria-label="Workflow steps">
      {ordered.map((step, index) => {
        const style = STATUS_STYLES[step.status];
        const isFailed = step.id === failedStep || step.status === "failed";
        return (
          <li
            key={step.id}
            className={`flex gap-4 rounded-lg border p-4 ${style.ring} ${
              isFailed ? "bg-red-50" : "bg-white"
            }`}
          >
            <div className="flex flex-col items-center pt-1">
              <span
                className={`h-3 w-3 rounded-full ${style.dot}`}
                aria-hidden
              />
              {index < ordered.length - 1 && (
                <span className="mt-1 h-full w-px flex-1 bg-zinc-200" />
              )}
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium text-zinc-900">
                  {STEP_LABELS[step.id] ?? step.id}
                </span>
                <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-xs text-zinc-600">
                  {style.label}
                </span>
                {step.duration_ms != null && step.duration_ms > 0 && (
                  <span className="text-xs text-zinc-500">
                    {step.duration_ms} ms
                  </span>
                )}
              </div>
              {step.error && (
                <p className="mt-2 text-sm text-red-700">{step.error}</p>
              )}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
