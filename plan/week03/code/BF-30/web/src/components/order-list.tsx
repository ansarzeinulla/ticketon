import type { IssuedTicket } from "@/lib/types";

/** The tickets of one order, one row each, with the code the door checks. */
export function OrderList({ tickets }: { tickets: IssuedTicket[] }) {
  return (
    <ul className="mt-3 divide-y divide-border-subtle rounded-lg border border-border-subtle">
      {tickets.map((ticket) => (
        <li key={ticket.id} className="flex items-center justify-between gap-4 px-4 py-3">
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{ticket.ticket_type_name}</p>
            <p className="font-mono text-xs text-foreground-muted">{ticket.ticket_code}</p>
          </div>
          <span className="shrink-0 rounded-full bg-success-soft px-2.5 py-0.5 text-xs font-medium capitalize text-success">
            {ticket.status.replace("_", " ")}
          </span>
        </li>
      ))}
    </ul>
  );
}
