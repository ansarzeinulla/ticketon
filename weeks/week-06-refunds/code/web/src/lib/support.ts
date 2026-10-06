import type { SupportStatus } from "@/lib/types";

/** Human labels for the support_case_category enum values. */
export const CATEGORY_LABELS: Record<string, string> = {
  ticket_delivery: "Ticket delivery",
  payment: "Payment",
  refund: "Refund",
  seating: "Seating",
  event_information: "Event information",
  check_in: "Check-in",
  account: "Account",
  technical: "Technical problem",
};

export function categoryLabel(category: string): string {
  return CATEGORY_LABELS[category] ?? category.replace(/_/g, " ");
}

export const STATUS_LABELS: Record<SupportStatus, string> = {
  open: "Open",
  in_progress: "In progress",
  waiting_for_customer: "Waiting for you",
  resolved: "Resolved",
};

/** Statuses an organizer can set, in the order they normally happen. */
export const ORGANIZER_STATUSES: SupportStatus[] = [
  "open",
  "in_progress",
  "waiting_for_customer",
  "resolved",
];

export const STATUS_STYLES: Record<SupportStatus, string> = {
  open: "bg-warning-soft text-warning",
  in_progress: "bg-brand-soft text-brand-strong",
  waiting_for_customer: "bg-surface-muted text-foreground-muted",
  resolved: "bg-success-soft text-success",
};

/** "14:32" for today, otherwise a short date and time. */
export function formatMessageTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;

  const today = new Date();
  const sameDay =
    date.getFullYear() === today.getFullYear() &&
    date.getMonth() === today.getMonth() &&
    date.getDate() === today.getDate();

  return new Intl.DateTimeFormat("en-GB", {
    ...(sameDay ? {} : { dateStyle: "medium" }),
    timeStyle: "short",
  }).format(date);
}
