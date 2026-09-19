/**
 * Fake gate answers, so the scanner screen can be built before the check-in
 * endpoint exists.
 *
 * Each entry is exactly what POST /events/{id}/check-in is going to return for
 * that kind of code: the body the Go API will send and its HTTP status. The
 * scanner is written against these, and this file is deleted the moment the
 * real endpoint answers - nothing in production ever reads it.
 */

export interface MockStats {
  issued: number;
  checked_in: number;
}

export interface MockAdmission {
  ticket_id: string;
  ticket_code: string;
  ticket_type_name: string;
  attendee_name: string;
  attendee_email: string;
  seat_label?: string;
  checked_in_at: string;
  stats: MockStats;
}

export type MockScanResponse =
  | { status: 200; body: { result: "valid"; check_in: MockAdmission } }
  | {
      status: 400 | 404 | 409;
      body: {
        error: {
          code: string;
          message: string;
          attendee_name?: string;
          checked_in_at?: string;
          stats?: MockStats;
        };
      };
    };

const stats: MockStats = { issued: 120, checked_in: 37 };

/** One code per outcome the scanner has to render. */
export const MOCK_SCANS: Record<string, MockScanResponse> = {
  // Green: a first scan.
  TKT_MOCKVALID000000001: {
    status: 200,
    body: {
      result: "valid",
      check_in: {
        ticket_id: "00000000-0000-4000-8000-000000000001",
        ticket_code: "BF-MOCK-0001",
        ticket_type_name: "Standard",
        attendee_name: "Nurlan Sagyndyk",
        attendee_email: "nurlan@example.kz",
        seat_label: "Orchestra, Row B, Seat 7",
        checked_in_at: "2026-10-14T18:02:11Z",
        stats: { ...stats, checked_in: stats.checked_in + 1 },
      },
    },
  },
  // Red: the same ticket a second time.
  TKT_MOCKUSED0000000001: {
    status: 409,
    body: {
      error: {
        code: "already_checked_in",
        message: "This ticket has already been used to enter.",
        attendee_name: "Aigerim Zhaksy",
        checked_in_at: "2026-10-14T17:48:30Z",
        stats,
      },
    },
  },
  // Red: a ticket for a different event.
  TKT_MOCKOTHEREVENT0001: {
    status: 409,
    body: {
      error: {
        code: "wrong_event",
        message: "This ticket is for AITU Freshers Evening, not this event.",
      },
    },
  },
  // Red: a promotional QR from a poster (SRS 4.14).
  CMP_MOCKSTUDENT15: {
    status: 400,
    body: {
      error: {
        code: "campaign_token",
        message: "This is a promotional campaign code, not an admission ticket.",
      },
    },
  },
};

/** What the gate would say about a code; anything not listed is unknown. */
export function mockScan(code: string): MockScanResponse {
  return (
    MOCK_SCANS[code.trim()] ?? {
      status: 404,
      body: {
        error: { code: "unknown_ticket", message: "This code does not match any ticket." },
      },
    }
  );
}
