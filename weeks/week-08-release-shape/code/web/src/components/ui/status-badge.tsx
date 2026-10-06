import type { EventStatus } from "@/lib/types";

const styles: Record<EventStatus, string> = {
  draft: "bg-surface-muted text-foreground-muted",
  published: "bg-success-soft text-success",
  unpublished: "bg-warning-soft text-warning",
  cancelled: "bg-danger-soft text-danger",
  completed: "bg-brand-soft text-brand-strong",
  suspended: "bg-danger-soft text-danger",
};

export function StatusBadge({ status }: { status: EventStatus }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${styles[status]}`}
    >
      {status}
    </span>
  );
}
