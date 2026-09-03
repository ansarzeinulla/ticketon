import { ticketPDFURL, ticketQRURL } from "@/lib/api";
import { getTranslations } from "@/lib/i18n/server";
import type { MessageKey } from "@/lib/i18n/dictionaries";
import type { IssuedTicket } from "@/lib/types";

/**
 * One issued ticket: the scannable code, its identifiers, and the download.
 *
 * A plain <img> rather than next/image on purpose. The QR is a per-ticket API
 * resource on another origin, served `no-store` because a ticket can be
 * cancelled; routing it through the image optimizer would cache and re-encode
 * it, and a re-encoded QR is a QR that might not scan.
 */
/** The ticket status, in words, in the visitor's language. */
const STATUS_KEY: Record<string, MessageKey> = {
  valid: "ticketCard.statusValid",
  checked_in: "ticketCard.statusCheckedIn",
  cancelled: "ticketCard.statusCancelled",
  refunded: "ticketCard.statusRefunded",
};

export async function TicketCard({ ticket }: { ticket: IssuedTicket }) {
  const { t } = await getTranslations();
  const cancelled = ticket.status !== "valid";
  const statusLabel = STATUS_KEY[ticket.status]
    ? t(STATUS_KEY[ticket.status])
    : ticket.status.replace("_", " ");

  return (
    <li className="flex flex-wrap items-start gap-5 rounded-xl border border-border-subtle bg-surface p-5">
      <div className="shrink-0 rounded-lg bg-white p-2">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={ticketQRURL(ticket.id)}
          alt={t("ticketCard.qrAlt", { code: ticket.ticket_code })}
          width={128}
          height={128}
          className={`h-32 w-32 ${cancelled ? "opacity-30" : ""}`}
        />
      </div>

      <div className="min-w-0 flex-1 space-y-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-medium">{ticket.ticket_type_name}</h3>
            <span
              className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
                cancelled
                  ? "bg-danger-soft text-danger"
                  : "bg-success-soft text-success"
              }`}
            >
              {statusLabel}
            </span>
          </div>
          <p className="mt-1 font-mono text-xs text-foreground-muted">
            {ticket.ticket_code}
          </p>
        </div>

        <div>
          <p className="text-xs text-foreground-muted">{t("ticketCard.admissionCode")}</p>
          <p className="mt-0.5 break-all font-mono text-xs">{ticket.qr_token}</p>
        </div>

        {!cancelled && (
          <a
            href={ticketPDFURL(ticket.id)}
            className="inline-flex items-center gap-2 rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-strong"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 20 20"
              fill="currentColor"
              className="h-4 w-4"
            >
              <path d="M10 2a1 1 0 0 1 1 1v7.586l2.293-2.293a1 1 0 1 1 1.414 1.414l-4 4a1 1 0 0 1-1.414 0l-4-4a1 1 0 1 1 1.414-1.414L9 10.586V3a1 1 0 0 1 1-1Z" />
              <path d="M3 14a1 1 0 0 1 1 1v1h12v-1a1 1 0 1 1 2 0v1a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2v-1a1 1 0 0 1 1-1Z" />
            </svg>
            {t("ticketCard.download")}
          </a>
        )}
      </div>
    </li>
  );
}
