#!/usr/bin/env bash
#
# BiletFlow API acceptance checks, over real HTTP with cURL.
#
# The same walk-through you would do by hand in Postman:
#   Phase 2: register -> login -> create and publish an event
#   Phase 4: add a ticket type -> buy through the simulated checkout ->
#            confirm the inventory moved -> confirm overselling is refused
#   Phase 5: download the A4 PDF ticket and its QR preview
#   Phase 7: create a promo code, redeem it, exhaust it, and confirm the gate
#            refuses the campaign QR
#   Phase 8: run a support conversation end to end, then suspend the event and
#            confirm checkout is blocked
#   Phase 9: compare every analytics figure against the rows, then duplicate
#            the event and confirm the copy is an empty draft
#   Phase 10: paid sales are refused until the activation checklist is done;
#            an organizer refunds an order and the tickets stop scanning
#   Phase 13: a basket reserves stock without selling it, expires on its own,
#            and is charged a processing fee; an event exports as .ics
#   Phase 12: a password is reset from an emailed token, an image is uploaded
#            and shown publicly, an attendee is found by name and checked in
#            without a QR, and an admin searches and exports a CSV report
#
# Usage:
#   ./api/scripts/smoke_test.sh                 against http://localhost:8080
#   API_URL=http://host:9000 ./api/scripts/smoke_test.sh
#
# Requires: curl, python3 (for reading JSON), and docker compose for the
# database check. Exits non-zero on the first failure.

set -uo pipefail

API_URL="${API_URL:-http://localhost:8080}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT" || exit 1

# shellcheck disable=SC1091
[ -f .env ] && set -a && . ./.env && set +a
DB_USER="${POSTGRES_USER:-biletflow}"
DB_NAME="${POSTGRES_DB:-biletflow}"

if [ -t 1 ]; then
    RED=$'\033[31m'; GREEN=$'\033[32m'; DIM=$'\033[2m'; BOLD=$'\033[1m'; OFF=$'\033[0m'
else
    RED=""; GREEN=""; DIM=""; BOLD=""; OFF=""
fi

failures=0
pass() { printf '%s\n' "${GREEN}PASS${OFF} $*"; }
fail() { printf '%s\n' "${RED}${BOLD}FAIL${OFF} $*"; failures=$((failures + 1)); }
info() { printf '%s\n' "${DIM}     $*${OFF}"; }

# json_get <json> <dotted.path> - prints the value, or nothing when absent.
json_get() {
    python3 -c '
import json, sys
try:
    data = json.loads(sys.argv[1])
except Exception:
    sys.exit(0)
for key in sys.argv[2].split("."):
    if not isinstance(data, dict) or key not in data:
        sys.exit(0)
    data = data[key]
print(data if not isinstance(data, (dict, list)) else json.dumps(data))
' "$1" "$2"
}

# request <method> <path> <token|-> <body|-> - sets HTTP_STATUS and HTTP_BODY.
request() {
    local method="$1" path="$2" token="${3:--}" body="${4:--}"
    local args=(-sS -o /tmp/biletflow_smoke_body -w '%{http_code}' -X "$method" "${API_URL}${path}")

    [ "$token" != "-" ] && args+=(-H "Authorization: Bearer ${token}")
    if [ "$body" != "-" ]; then
        args+=(-H 'Content-Type: application/json' -d "$body")
    fi

    HTTP_STATUS="$(curl "${args[@]}" 2>/dev/null)"
    HTTP_BODY="$(cat /tmp/biletflow_smoke_body 2>/dev/null)"
    rm -f /tmp/biletflow_smoke_body
}

expect_status() {
    local want="$1" label="$2"
    if [ "$HTTP_STATUS" = "$want" ]; then
        pass "$label (HTTP $HTTP_STATUS)"
        return 0
    fi
    fail "$label - got HTTP ${HTTP_STATUS:-none}, want $want"
    info "response: $HTTP_BODY"
    return 1
}

printf '%s\n' "${BOLD}BiletFlow API acceptance checks (Phases 2-13)${OFF}  ->  $API_URL"
printf '%s\n' "-----------------------------------------------------------"

# --- the API must be up ------------------------------------------------------
request GET /health - -
if ! expect_status 200 "00  API is reachable and healthy"; then
    printf '\n%s\n' "Start the API first:  make api-run    (and make up for the database)"
    exit 1
fi

# A unique address per run, so the script can be run repeatedly.
STAMP="$(date +%s)"
EMAIL="smoke.${STAMP}@biletflow.test"
PASSWORD="correct horse battery"

# --- criterion 1: register ---------------------------------------------------
request POST /api/v1/auth/register - \
    "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}"
expect_status 201 "01  register with email and password" || exit 1

USER_ID="$(json_get "$HTTP_BODY" user.id)"
if [ -z "$USER_ID" ]; then
    fail "01  registration response contains no user id"
    exit 1
fi
info "user id: $USER_ID"

# The same address must not register twice.
request POST /api/v1/auth/register - \
    "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}"
expect_status 409 "01b registering the same email again is rejected"

# --- criterion 2: login ------------------------------------------------------
request POST /api/v1/auth/login - \
    "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}"
expect_status 200 "02  login returns a token" || exit 1

TOKEN="$(json_get "$HTTP_BODY" access_token)"
if [ -z "$TOKEN" ]; then
    fail "02  login response contains no access_token"
    exit 1
fi
info "token: ${TOKEN:0:32}..."

# A token is only "valid" if it actually authorises a request.
request GET /api/v1/auth/me "$TOKEN" -
expect_status 200 "02b the token authenticates a protected request"

request POST /api/v1/auth/login - \
    "{\"email\":\"${EMAIL}\",\"password\":\"wrong password\"}"
expect_status 401 "02c the wrong password is rejected"

# --- criterion 3: create an event -------------------------------------------
STARTS_AT="$(python3 -c '
import datetime
start = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=30)
print(start.replace(microsecond=0).isoformat())')"
ENDS_AT="$(python3 -c '
import datetime
end = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=30, hours=3)
print(end.replace(microsecond=0).isoformat())')"

EVENT_BODY=$(cat <<JSON
{
  "title": "Smoke Test Concert ${STAMP}",
  "description": "Created by api/scripts/smoke_test.sh",
  "category": "music",
  "venue_name": "Almaty Demo Hall",
  "venue_address": "Abay Avenue 44, Almaty",
  "starts_at": "${STARTS_AT}",
  "ends_at": "${ENDS_AT}",
  "timezone": "Asia/Almaty",
  "capacity": 250
}
JSON
)

request POST /api/v1/events "$TOKEN" "$EVENT_BODY"
expect_status 201 "03  POST /events with the token returns 201 Created" || exit 1

EVENT_ID="$(json_get "$HTTP_BODY" event.id)"
if [ -z "$EVENT_ID" ]; then
    fail "03  the create response contains no event id"
    exit 1
fi
info "event id: $EVENT_ID"

# The row must be in PostgreSQL, checked outside the API.
DB_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT title || ' | ' || status || ' | ' || organizer_id FROM events WHERE id = '${EVENT_ID}';" 2>&1)"

if printf '%s' "$DB_ROW" | grep -q "Smoke Test Concert ${STAMP}"; then
    pass "03b the event is physically in the events table"
    info "row: $(printf '%s' "$DB_ROW" | tr -d '\n')"
else
    fail "03b the event was not found in PostgreSQL"
    info "psql said: $DB_ROW"
fi

if printf '%s' "$DB_ROW" | grep -q "$USER_ID"; then
    pass "03c the event is owned by the registered user"
else
    fail "03c the event's organizer_id does not match the registered user"
fi

# --- authorisation is actually enforced --------------------------------------
request POST /api/v1/events - "$EVENT_BODY"
expect_status 401 "04  POST /events without a token is rejected"

request GET /api/v1/events/"$EVENT_ID" - -
expect_status 404 "05  an unpublished draft is not publicly readable"

request POST /api/v1/events/"$EVENT_ID"/publish "$TOKEN" -
expect_status 200 "06  the organizer can publish the event"

request GET /api/v1/events/"$EVENT_ID" - -
expect_status 200 "07  the published event is publicly readable"

# --- Phase 4: ticket types and the simulated checkout ------------------------

request POST /api/v1/events/"$EVENT_ID"/ticket-types "$TOKEN" \
    '{"name":"General Admission","price_kzt":"5000","quantity_total":5,"max_per_order":10}'
expect_status 201 "08  organizer creates 5 tickets at 5000 KZT" || exit 1

TICKET_TYPE_ID="$(json_get "$HTTP_BODY" ticket_type.id)"
if [ -z "$TICKET_TYPE_ID" ]; then
    fail "08  the ticket type response contains no id"
    exit 1
fi
info "ticket type id: $TICKET_TYPE_ID"

# --- Phase 10: paid-sales activation gates the sale (SRS 4.5) ----------------
#
# The ticket type above costs money, so nothing can be sold until the organizer
# has been through the activation checklist. These checks run here, before the
# first purchase, because that is exactly the order an organizer meets them in.

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Too Early\",\"buyer_email\":\"early.${STAMP}@biletflow.test\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 403 "08b a paid ticket cannot be bought before activation"
if [ "$(json_get "$HTTP_BODY" error.code)" = "paid_sales_not_active" ]; then
    pass "08c the refusal is an explicit paid_sales_not_active"
else
    fail "08c refusal code is $(json_get "$HTTP_BODY" error.code), want paid_sales_not_active"
fi

TICKETS_ISSUED="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT count(*) FROM tickets WHERE event_id = '${EVENT_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$TICKETS_ISSUED" = "0" ]; then
    pass "08d the blocked purchase issued no ticket"
else
    fail "08d ${TICKETS_ISSUED} ticket(s) exist after a blocked purchase"
fi

request GET /api/v1/events/"$EVENT_ID"/activation "$TOKEN" -
expect_status 200 "08e the organizer reads the activation checklist"
if [ "$(json_get "$HTTP_BODY" activation.status)" = "not_started" ] &&
   [ "$(json_get "$HTTP_BODY" activation.required_for_sales)" = "True" ]; then
    pass "08f it is not_started and required, because this event sells paid tickets"
else
    fail "08f activation is $(json_get "$HTTP_BODY" activation.status), required=$(json_get "$HTTP_BODY" activation.required_for_sales)"
fi

# Half a checklist is not a checklist.
request POST /api/v1/events/"$EVENT_ID"/activation "$TOKEN" \
    '{"accept_terms":true,"confirm_identity":true}'
expect_status 200 "09  two of the four steps are recorded"
if [ "$(json_get "$HTTP_BODY" activation.is_active)" = "False" ]; then
    pass "09b a partial checklist does not open sales"
else
    fail "09b half a checklist activated the event"
fi

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Still Early\",\"buyer_email\":\"still.${STAMP}@biletflow.test\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 403 "09c buying is still refused halfway through the checklist"

request POST /api/v1/events/"$EVENT_ID"/activation "$TOKEN" \
    '{"confirm_payout":true,"pay_activation_fee":true}'
expect_status 200 "09d the organizer completes the checklist"
if [ "$(json_get "$HTTP_BODY" activation.is_active)" = "True" ] &&
   [ "$(json_get "$HTTP_BODY" activation.status)" = "active" ]; then
    pass "09e paid sales are now active"
else
    fail "09e activation is $(json_get "$HTTP_BODY" activation.status), want active"
fi

ACTIVATION_FEE="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT p.amount_kzt || '/' || p.is_simulated
       FROM payments p JOIN paid_sales_activations a ON a.activation_payment_id = p.id
      WHERE a.event_id = '${EVENT_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$ACTIVATION_FEE" = "5000.00/true" ]; then
    pass "09f PostgreSQL records the 5000.00 KZT activation fee as simulated"
else
    fail "09f the activation fee row is ${ACTIVATION_FEE:-missing}, want 5000.00/true"
fi

PAID_SALES_FLAG="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT paid_sales_enabled FROM events WHERE id = '${EVENT_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$PAID_SALES_FLAG" = "t" ]; then
    pass "09g events.paid_sales_enabled was set in step with the activation"
else
    fail "09g paid_sales_enabled is ${PAID_SALES_FLAG:-nothing}, want t"
fi



# The slug is generated from the title, so it is read back rather than guessed.
request GET /api/v1/events/"$EVENT_ID" "$TOKEN" -
SLUG="$(json_get "$HTTP_BODY" event.slug)"
info "slug: $SLUG"

request GET /api/v1/public/events/"$SLUG" - -
expect_status 200 "09  the public event page lists the ticket type"

REMAINING="$(python3 -c '
import json, sys
data = json.loads(sys.argv[1])
types = data.get("ticket_types", [])
print(types[0]["quantity_remaining"] if types else "")
' "$HTTP_BODY")"
if [ "$REMAINING" = "5" ]; then
    pass "09b the public page reports 5 remaining"
else
    fail "09b the public page reports ${REMAINING:-none} remaining, want 5"
fi

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Nurlan Amanov\",\"buyer_email\":\"nurlan.${STAMP}@biletflow.test\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":2}]}"
expect_status 201 "10  an attendee buys 2 tickets with the simulated checkout" || exit 1

ORDER_ID="$(json_get "$HTTP_BODY" order.id)"
ORDER_STATUS="$(json_get "$HTTP_BODY" order.status)"
ORDER_TOTAL="$(json_get "$HTTP_BODY" order.total_kzt)"

if [ "$ORDER_STATUS" = "paid" ] && [ "$ORDER_TOTAL" = "10350.00" ]; then
    pass "10b the order is paid for 10350.00 KZT (10000 + the 3.5% processing charge)"
    info "order id: $ORDER_ID"
else
    fail "10b order is ${ORDER_STATUS:-none} for ${ORDER_TOTAL:-none}, want paid / 10350.00"
fi

# The inventory is checked outside the API, straight from PostgreSQL.
INVENTORY="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT quantity_sold || '/' || quantity_total FROM ticket_types WHERE id = '${TICKET_TYPE_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$INVENTORY" = "2/5" ]; then
    pass "11  PostgreSQL confirms quantity_sold is 2 with 3 remaining"
else
    fail "11  ticket_types reports ${INVENTORY:-nothing}, want 2/5"
fi

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Aigerim Serik\",\"buyer_email\":\"aigerim.${STAMP}@biletflow.test\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":4}]}"
expect_status 409 "12  buying 4 more is rejected: only 3 remain"

REJECT_CODE="$(json_get "$HTTP_BODY" error.code)"
if [ "$REJECT_CODE" = "insufficient_inventory" ]; then
    pass "12b the rejection is an explicit insufficient_inventory"
else
    fail "12b rejection code is ${REJECT_CODE:-none}, want insufficient_inventory"
fi

INVENTORY_AFTER="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT quantity_sold || '/' || quantity_total FROM ticket_types WHERE id = '${TICKET_TYPE_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$INVENTORY_AFTER" = "2/5" ]; then
    pass "12c the rejected order changed no inventory"
else
    fail "12c inventory moved to ${INVENTORY_AFTER:-nothing} after a rejected order"
fi

request GET /api/v1/orders/"$ORDER_ID" - -
expect_status 200 "13  the order confirmation is retrievable by id"

# --- Cart holds, fees and calendar export (SRS 4.6, 3.3, 4.11) ---------------

# The event still has stock at this point in the run.
request POST /api/v1/events/"$EVENT_ID"/holds - \
    "{\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 201 "62  an attendee reserves a ticket without buying it"

HOLD_ORDER="$(json_get "$HTTP_BODY" hold.order_id)"
HOLD_FEE="$(json_get "$HTTP_BODY" hold.estimated_processing_fee_kzt)"
HOLD_TOTAL="$(json_get "$HTTP_BODY" hold.estimated_total_kzt)"

if [ "$HOLD_FEE" = "175.00" ] && [ "$HOLD_TOTAL" = "5175.00" ]; then
    pass "62b the basket is quoted 5000 plus the 3.5 percent charge"
else
    fail "62b quote is ${HOLD_FEE:-none} / ${HOLD_TOTAL:-none}, want 175.00 / 5175.00"
fi

RESERVED_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (SELECT quantity_reserved FROM ticket_types WHERE id = '${TICKET_TYPE_ID}')
         || '/' || (SELECT status FROM orders WHERE id = '${HOLD_ORDER}')
         || '/' || (SELECT (reserved_until IS NOT NULL)::text FROM orders WHERE id = '${HOLD_ORDER}');" \
    2>&1 | tr -d '\r\n ')"
if [ "$RESERVED_ROW" = "1/pending/true" ]; then
    pass "63  PostgreSQL reserved the ticket without selling it"
else
    fail "63  the reservation reads ${RESERVED_ROW:-nothing}, want 1/pending/true"
fi

request GET /api/v1/orders/"$HOLD_ORDER"/hold - -
expect_status 200 "63b a reloaded page can read its basket back"

request DELETE /api/v1/orders/"$HOLD_ORDER"/hold - -
expect_status 200 "64  releasing the basket puts the ticket back"

RELEASED="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT quantity_reserved FROM ticket_types WHERE id = '${TICKET_TYPE_ID}';" \
    2>&1 | tr -d '\r\n ')"
if [ "$RELEASED" = "0" ]; then
    pass "64b quantity_reserved is back to zero"
else
    fail "64b quantity_reserved is ${RELEASED:-nothing} after releasing, want 0"
fi

# An abandoned basket expires on its own (SRS 4.6). The clock is moved rather
# than waited on: a check that sleeps fifteen minutes is a check nobody runs.
request POST /api/v1/events/"$EVENT_ID"/holds - \
    "{\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 201 "65  a second basket is reserved, then abandoned"
ABANDONED="$(json_get "$HTTP_BODY" hold.order_id)"

docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -qtAX -c \
    "UPDATE orders SET reserved_until = now() - interval '1 minute'
      WHERE id = '${ABANDONED}';" >/dev/null 2>&1

request POST /api/v1/orders/"$ABANDONED"/confirm - \
    "{\"buyer_name\":\"Too Late\",\"buyer_email\":\"late.${STAMP}@biletflow.test\"}"
expect_status 409 "65b confirming an expired basket is refused"
if [ "$(json_get "$HTTP_BODY" error.code)" = "hold_expired" ]; then
    pass "65c the refusal says the reservation expired"
else
    fail "65c refusal code is $(json_get "$HTTP_BODY" error.code), want hold_expired"
fi

EXPIRED_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (SELECT status FROM orders WHERE id = '${ABANDONED}')
         || '/' || (SELECT quantity_reserved FROM ticket_types WHERE id = '${TICKET_TYPE_ID}');" \
    2>&1 | tr -d '\r\n ')"
if [ "$EXPIRED_ROW" = "expired/0" ]; then
    pass "66  the abandoned basket expired and its ticket went back on sale"
else
    fail "66  the abandoned basket reads ${EXPIRED_ROW:-nothing}, want expired/0"
fi

# A basket that is paid for becomes a sale, not an addition to one.
request POST /api/v1/events/"$EVENT_ID"/holds - \
    "{\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 201 "67  a third basket is reserved"
PAID_HOLD="$(json_get "$HTTP_BODY" hold.order_id)"

request POST /api/v1/orders/"$PAID_HOLD"/confirm - \
    "{\"buyer_name\":\"Two Step Buyer\",\"buyer_email\":\"twostep.${STAMP}@biletflow.test\"}"
expect_status 200 "67b the basket is paid for and the tickets issued"

CONVERTED="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (SELECT quantity_reserved FROM ticket_types WHERE id = '${TICKET_TYPE_ID}')
         || '/' || (SELECT status FROM orders WHERE id = '${PAID_HOLD}')
         || '/' || (SELECT count(*) FROM tickets WHERE order_id = '${PAID_HOLD}');" \
    2>&1 | tr -d '\r\n ')"
if [ "$CONVERTED" = "0/paid/1" ]; then
    pass "68  the reservation became a sale: 0 reserved, order paid, 1 ticket"
else
    fail "68  after confirming, the rows read ${CONVERTED:-nothing}, want 0/paid/1"
fi

FEE_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT subtotal_kzt || '/' || processing_fee_kzt || '/' || total_kzt
       FROM orders WHERE id = '${PAID_HOLD}';" 2>&1 | tr -d '\r\n ')"
if [ "$FEE_ROW" = "5000.00/175.00/5175.00" ]; then
    pass "68b the processing charge was applied, and orders_total_math_chk accepted it"
else
    fail "68b the money reads ${FEE_ROW:-nothing}, want 5000.00/175.00/5175.00"
fi

# --- calendar export (SRS 4.11) ----------------------------------------------

CAL_HEADERS="$(curl -sS -D - -o "$(mktemp -t biletflow_cal).ics" \
    "${API_URL}/api/v1/events/${EVENT_ID}/calendar.ics" 2>/dev/null)"
if printf '%s' "$CAL_HEADERS" | grep -qi "content-type: text/calendar"; then
    pass "69  the event exports as an iCalendar file"
else
    fail "69  the calendar Content-Type is not text/calendar"
fi

CAL_BODY="$(curl -sS "${API_URL}/api/v1/events/${EVENT_ID}/calendar.ics" 2>/dev/null)"
if printf '%s' "$CAL_BODY" | grep -q "BEGIN:VCALENDAR" &&
   printf '%s' "$CAL_BODY" | grep -q "BEGIN:VEVENT" &&
   printf '%s' "$CAL_BODY" | grep -q "DTSTART;TZID=Asia/Almaty:"; then
    pass "69b it carries a VEVENT in the event's own timezone"
else
    fail "69b the calendar file is missing its VEVENT or timezone"
fi

if printf '%s' "$CAL_BODY" | grep -q "UID:${EVENT_ID}@biletflow.kz"; then
    pass "69c the UID is stable, so a re-download replaces the entry"
else
    fail "69c the calendar UID is not the event id"
fi

# --- Phase 5: QR codes and printable PDF tickets -----------------------------

TICKET_ID="$(python3 -c '
import json, sys
data = json.loads(sys.argv[1])
tickets = data.get("tickets", [])
print(tickets[0]["id"] if tickets else "")
' "$HTTP_BODY")"
TICKET_TOKEN="$(python3 -c '
import json, sys
data = json.loads(sys.argv[1])
tickets = data.get("tickets", [])
print(tickets[0]["qr_token"] if tickets else "")
' "$HTTP_BODY")"

if printf '%s' "$TICKET_TOKEN" | grep -qE '^TKT_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then
    pass "14  the admission token is TKT_<uuid>"
    info "token: $TICKET_TOKEN"
else
    fail "14  admission token is '${TICKET_TOKEN:-none}', want TKT_<uuid>"
fi

PDF_FILE="$(mktemp -t biletflow-ticket).pdf"
PDF_HEADERS="$(mktemp)"
curl -sS -D "$PDF_HEADERS" -o "$PDF_FILE" "${API_URL}/api/v1/tickets/${TICKET_ID}/pdf" 2>/dev/null

if grep -qi 'Content-Type: application/pdf' "$PDF_HEADERS"; then
    pass "15  the ticket PDF is served as application/pdf"
else
    fail "15  the PDF response has the wrong content type"
fi

if grep -qi 'Content-Disposition: attachment' "$PDF_HEADERS"; then
    pass "15b it is sent as an attachment, so the browser downloads it"
else
    fail "15b the PDF is not sent as an attachment"
fi

if head -c 5 "$PDF_FILE" | grep -q '%PDF-'; then
    pass "16  the file is a real PDF ($(wc -c < "$PDF_FILE" | tr -d ' ') bytes)"
else
    fail "16  the downloaded file is not a PDF"
fi

if grep -aq '595.28' "$PDF_FILE" && grep -aq '841.89' "$PDF_FILE"; then
    pass "17  the page is A4 (595.28 x 841.89 pt)"
else
    fail "17  the page is not A4"
fi

request GET /api/v1/tickets/"$TICKET_ID" - -
expect_status 200 "18  the ticket exposes its delivery links"

QR_HEADERS="$(mktemp)"
QR_FILE="$(mktemp -t biletflow-qr).png"
curl -sS -D "$QR_HEADERS" -o "$QR_FILE" "${API_URL}/api/v1/tickets/${TICKET_ID}/qr.png" 2>/dev/null
if grep -qi 'Content-Type: image/png' "$QR_HEADERS" && head -c 4 "$QR_FILE" | grep -q 'PNG'; then
    pass "19  the QR preview is served as a PNG"
else
    fail "19  the QR preview is not a PNG"
fi

rm -f "$PDF_FILE" "$PDF_HEADERS" "$QR_FILE" "$QR_HEADERS"

# --- Phase 7: promo codes and campaign QR codes ------------------------------

request POST /api/v1/events/"$EVENT_ID"/campaigns "$TOKEN" \
    "{\"name\":\"Smoke Promo\",\"code\":\"SMOKE${STAMP}\",\"discount_type\":\"percentage\",\"discount_value\":\"20\",\"max_redemptions\":1}"
expect_status 201 "20  organizer creates a 20% promo code, limit 1" || exit 1

CAMPAIGN_ID="$(json_get "$HTTP_BODY" campaign.id)"
CAMPAIGN_TOKEN="$(json_get "$HTTP_BODY" campaign.qr_token)"
CAMPAIGN_URL="$(json_get "$HTTP_BODY" campaign.campaign_url)"

if printf '%s' "$CAMPAIGN_TOKEN" | grep -qE '^CMP_[0-9a-f-]{36}$'; then
    pass "20b the campaign token is CMP_<uuid>"
    info "token: $CAMPAIGN_TOKEN"
else
    fail "20b campaign token is '${CAMPAIGN_TOKEN:-none}', want CMP_<uuid>"
fi

# SRS 4.14: the link carries an opaque token, never a discount value.
if printf '%s' "$CAMPAIGN_URL" | grep -q "$CAMPAIGN_TOKEN" &&
   ! printf '%s' "$CAMPAIGN_URL" | grep -qi 'discount\|percent'; then
    pass "20c the campaign link carries the token and no discount value"
else
    fail "20c campaign link looks wrong: $CAMPAIGN_URL"
fi

request POST /api/v1/events/"$EVENT_ID"/promo/preview - \
    "{\"campaign_token\":\"${CAMPAIGN_TOKEN}\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 200 "21  the campaign token prices a basket"

PREVIEW_TOTAL="$(json_get "$HTTP_BODY" total_kzt)"
PREVIEW_DISCOUNT="$(json_get "$HTTP_BODY" discount_kzt)"
if [ "$PREVIEW_DISCOUNT" = "1000.00" ] && [ "$PREVIEW_TOTAL" = "4000.00" ]; then
    pass "21b 5000 KZT less 20% is 4000 KZT"
else
    fail "21b preview says ${PREVIEW_DISCOUNT:-none} off for ${PREVIEW_TOTAL:-none}, want 1000.00 / 4000.00"
fi

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Promo Buyer\",\"buyer_email\":\"promo.${STAMP}@biletflow.test\",\"campaign_token\":\"${CAMPAIGN_TOKEN}\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 201 "22  an attendee checks out with the campaign discount"

PROMO_ORDER_TOTAL="$(json_get "$HTTP_BODY" order.total_kzt)"
if [ "$PROMO_ORDER_TOTAL" = "4140.00" ]; then
    pass "22b the order is stored at the discounted 4140.00 KZT"
else
    fail "22b order total is ${PROMO_ORDER_TOTAL:-none}, want 4140.00 (4000 + the 3.5% processing charge)"
fi

REDEEMED="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT redemption_count FROM campaigns WHERE id = '${CAMPAIGN_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$REDEEMED" = "1" ]; then
    pass "23  PostgreSQL confirms redemption_count is 1"
else
    fail "23  redemption_count is ${REDEEMED:-nothing}, want 1"
fi

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Late Buyer\",\"buyer_email\":\"late.${STAMP}@biletflow.test\",\"promo_code\":\"SMOKE${STAMP}\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 409 "24  a second use of the single-use code is refused"

if [ "$(json_get "$HTTP_BODY" error.code)" = "promo_exhausted" ]; then
    pass "24b the refusal is an explicit promo_exhausted"
else
    fail "24b refusal code is $(json_get "$HTTP_BODY" error.code), want promo_exhausted"
fi

# SRS 4.14: the gate must never accept a promotional code.
request POST /api/v1/events/"$EVENT_ID"/check-in "$TOKEN" \
    "{\"qr_token\":\"${CAMPAIGN_TOKEN}\"}"
expect_status 400 "25  the gate refuses the bare campaign token"

request POST /api/v1/events/"$EVENT_ID"/check-in "$TOKEN" \
    "{\"qr_token\":\"${CAMPAIGN_URL}\"}"
expect_status 400 "25b the gate refuses the campaign QR link"

if [ "$(json_get "$HTTP_BODY" error.code)" = "campaign_token" ]; then
    pass "25c the refusal names it as a campaign code"
else
    fail "25c refusal code is $(json_get "$HTTP_BODY" error.code), want campaign_token"
fi

# --- Phase 8: support chat and platform moderation ---------------------------

# The buyer needs an account to hold a conversation against; they register with
# the address the smoke-test order was bought under.
request POST /api/v1/auth/register - \
    "{\"email\":\"buyer.${STAMP}@biletflow.test\",\"password\":\"buyer1234567\"}"
expect_status 201 "26  the ticket buyer registers an account" || exit 1
BUYER_TOKEN="$(json_get "$HTTP_BODY" access_token)"

request POST /api/v1/events/"$EVENT_ID"/checkout "$BUYER_TOKEN" \
    "{\"buyer_name\":\"Support Buyer\",\"buyer_email\":\"buyer.${STAMP}@biletflow.test\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 201 "26b they buy a ticket" || exit 1
SUPPORT_ORDER_ID="$(json_get "$HTTP_BODY" order.id)"

request POST /api/v1/support/cases "$BUYER_TOKEN" \
    "{\"category\":\"event_information\",\"subject\":\"Parking\",\"message\":\"Where is parking?\",\"order_id\":\"${SUPPORT_ORDER_ID}\"}"
expect_status 201 "27  the attendee opens a support case on their order" || exit 1

CASE_ID="$(json_get "$HTTP_BODY" case.id)"
if [ "$(json_get "$HTTP_BODY" case.status)" = "open" ]; then
    pass "27b the case starts Open"
else
    fail "27b case status is $(json_get "$HTTP_BODY" case.status), want open"
fi

# The order's event is captured automatically, which is how the organizer finds it.
request GET /api/v1/events/"$EVENT_ID"/support/cases "$TOKEN" -
expect_status 200 "28  the case appears in the organizer's inbox"

request POST /api/v1/support/cases/"$CASE_ID"/messages "$TOKEN" \
    '{"message":"Parking is in Zone B"}'
expect_status 201 "29  the organizer replies"

if [ "$(json_get "$HTTP_BODY" case.status)" = "in_progress" ]; then
    pass "29b replying moves the case to In Progress"
else
    fail "29b case status is $(json_get "$HTTP_BODY" case.status), want in_progress"
fi

request PATCH /api/v1/support/cases/"$CASE_ID" "$TOKEN" '{"status":"resolved"}'
expect_status 200 "30  the organizer marks it Resolved"

request GET /api/v1/support/cases/"$CASE_ID" "$BUYER_TOKEN" -
expect_status 200 "31  the attendee re-reads the case"

ATTENDEE_STATUS="$(json_get "$HTTP_BODY" case.status)"
REPLY_SEEN="$(python3 -c '
import json, sys
data = json.loads(sys.argv[1])
print("yes" if any(m["body"] == "Parking is in Zone B" for m in data.get("messages", [])) else "no")
' "$HTTP_BODY")"

if [ "$ATTENDEE_STATUS" = "resolved" ] && [ "$REPLY_SEEN" = "yes" ]; then
    pass "31b they see the reply and the Resolved status"
else
    fail "31b attendee sees status ${ATTENDEE_STATUS:-none}, reply seen: ${REPLY_SEEN:-no}"
fi

# --- platform moderation -----------------------------------------------------

request POST /api/v1/admin/events/"$EVENT_ID"/suspend "$TOKEN" -
expect_status 403 "32  an organizer cannot suspend their own event"

request POST /api/v1/admin/events/"$EVENT_ID"/suspend - -
expect_status 401 "32b an anonymous caller cannot suspend an event"

# Promote the smoke-test user to platform admin, the way an operator would.
docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -qtAX -c \
    "INSERT INTO user_roles (user_id, role) VALUES ('${USER_ID}', 'platform_admin')
     ON CONFLICT DO NOTHING;" >/dev/null 2>&1

request POST /api/v1/admin/events/"$EVENT_ID"/suspend "$TOKEN" \
    '{"reason":"Reported by attendees"}'
expect_status 200 "33  a platform admin suspends the event"

SUSPENDED_STATUS="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT status FROM events WHERE id = '${EVENT_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$SUSPENDED_STATUS" = "suspended" ]; then
    pass "33b PostgreSQL records the event as suspended"
else
    fail "33b event status is ${SUSPENDED_STATUS:-nothing}, want suspended"
fi

TICKETS_BEFORE="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT count(*) FROM tickets WHERE event_id = '${EVENT_ID}';" 2>&1 | tr -d '\r\n ')"

request POST /api/v1/events/"$EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Blocked Buyer\",\"buyer_email\":\"blocked.${STAMP}@biletflow.test\",\"items\":[{\"ticket_type_id\":\"${TICKET_TYPE_ID}\",\"quantity\":1}]}"
expect_status 403 "34  checkout on a suspended event is refused"

if [ "$(json_get "$HTTP_BODY" error.code)" = "event_suspended" ]; then
    pass "34b the refusal is an explicit event_suspended"
else
    fail "34b refusal code is $(json_get "$HTTP_BODY" error.code), want event_suspended"
fi

TICKETS_AFTER="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT count(*) FROM tickets WHERE event_id = '${EVENT_ID}';" 2>&1 | tr -d '\r\n ')"
if [ "$TICKETS_AFTER" = "$TICKETS_BEFORE" ]; then
    pass "34c no ticket was issued (${TICKETS_AFTER} before and after)"
else
    fail "34c tickets went from ${TICKETS_BEFORE} to ${TICKETS_AFTER}"
fi

# The page stays reachable so holders learn what happened.
request GET /api/v1/public/events/"$SLUG" - -
expect_status 200 "35  the public page still resolves for a suspended event"
if [ "$(json_get "$HTTP_BODY" suspended)" = "True" ]; then
    pass "35b it is flagged suspended so the banner shows"
else
    fail "35b public page suspended flag is $(json_get "$HTTP_BODY" suspended), want True"
fi

# --- Phase 9: organizer analytics and event duplication ----------------------
#
# Every figure the analytics endpoint reports is compared against the same
# number computed straight from the rows, so the page can never drift from
# the database without this failing.

request GET /api/v1/events/"$EVENT_ID"/analytics "$TOKEN" -
expect_status 200 "36  the organizer reads the analytics for the event"

A_SOLD="$(json_get "$HTTP_BODY" analytics.tickets_sold)"
A_REVENUE="$(json_get "$HTTP_BODY" analytics.gross_revenue_kzt)"
A_CHECKED="$(json_get "$HTTP_BODY" analytics.checked_in)"
A_PCT_SOLD="$(json_get "$HTTP_BODY" analytics.percentage_sold)"
A_PCT_IN="$(json_get "$HTTP_BODY" analytics.check_in_percentage)"
A_ORDERS="$(json_get "$HTTP_BODY" analytics.orders_count)"

DB_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (SELECT count(*) FROM tickets WHERE event_id = '${EVENT_ID}'
               AND status IN ('valid','checked_in'))
         || '|' || (SELECT to_char(coalesce(sum(total_kzt),0), 'FM999999999.00')
                      FROM orders WHERE event_id = '${EVENT_ID}' AND status = 'paid')
         || '|' || (SELECT count(*) FROM tickets WHERE event_id = '${EVENT_ID}'
                      AND status = 'checked_in')
         || '|' || (SELECT count(*) FROM orders WHERE event_id = '${EVENT_ID}'
                      AND status = 'paid');" 2>&1 | tr -d '\r\n ')"
DB_SOLD="${DB_ROW%%|*}"; REST="${DB_ROW#*|}"
DB_REVENUE="${REST%%|*}"; REST="${REST#*|}"
DB_CHECKED="${REST%%|*}"; DB_ORDERS="${REST#*|}"

if [ "$A_SOLD" = "$DB_SOLD" ]; then
    pass "36b tickets sold matches PostgreSQL ($A_SOLD)"
else
    fail "36b analytics says ${A_SOLD:-none} sold, PostgreSQL says ${DB_SOLD:-none}"
fi

if [ "$A_REVENUE" = "$DB_REVENUE" ]; then
    pass "36c KZT revenue matches PostgreSQL ($A_REVENUE)"
else
    fail "36c analytics says ${A_REVENUE:-none} KZT, PostgreSQL says ${DB_REVENUE:-none}"
fi

if [ "$A_CHECKED" = "$DB_CHECKED" ]; then
    pass "36d check-ins match PostgreSQL ($A_CHECKED)"
else
    fail "36d analytics says ${A_CHECKED:-none} checked in, PostgreSQL says ${DB_CHECKED:-none}"
fi

if [ "$A_ORDERS" = "$DB_ORDERS" ]; then
    pass "36e paid order count matches PostgreSQL ($A_ORDERS)"
else
    fail "36e analytics says ${A_ORDERS:-none} orders, PostgreSQL says ${DB_ORDERS:-none}"
fi

# The percentages are derived, so they are recomputed rather than trusted.
PCT_EXPECTED="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT round(100.0 * ${DB_SOLD} / nullif((SELECT sum(quantity_total)
                FROM ticket_types WHERE event_id = '${EVENT_ID}'), 0), 1)
         || '|' || coalesce(round(100.0 * ${DB_CHECKED} / nullif(${DB_SOLD}, 0), 1)::text, '0.0');" \
    2>&1 | tr -d '\r\n ')"
# JSON renders 80.0 as 80, so the two are compared as numbers, not strings.
if python3 -c '
import sys
got = [float(x) for x in sys.argv[1].split("|")]
want = [float(x) for x in sys.argv[2].split("|")]
sys.exit(0 if got == want else 1)
' "${A_PCT_SOLD}|${A_PCT_IN}" "$PCT_EXPECTED" 2>/dev/null; then
    pass "36f sold and check-in percentages are ${A_PCT_SOLD}% and ${A_PCT_IN}%, as computed from the rows"
else
    fail "36f percentages are ${A_PCT_SOLD}|${A_PCT_IN}, computed from rows: ${PCT_EXPECTED}"
fi

# Analytics belong to the owner only.
request GET /api/v1/events/"$EVENT_ID"/analytics - -
expect_status 401 "37  analytics without a token is refused"

request GET /api/v1/events/"$EVENT_ID"/timeline "$TOKEN" -
expect_status 200 "38  the event history is readable"
TIMELINE_COUNT="$(python3 -c '
import json, sys
try:
    print(len(json.loads(sys.argv[1]).get("entries", [])))
except Exception:
    print(0)
' "$HTTP_BODY")"
if [ "${TIMELINE_COUNT:-0}" -gt 0 ]; then
    pass "38b the history has ${TIMELINE_COUNT} recorded entries"
else
    fail "38b the history came back empty"
fi

# --- duplication: config is copied, sales history is not ---------------------

request POST /api/v1/events/"$EVENT_ID"/duplicate "$TOKEN" -
expect_status 201 "39  the organizer duplicates the event" || exit 1

DUP_ID="$(json_get "$HTTP_BODY" event.id)"
DUP_STATUS="$(json_get "$HTTP_BODY" event.status)"
info "duplicate id: $DUP_ID"

if [ "$DUP_STATUS" = "draft" ]; then
    pass "39b the copy comes back as a draft"
else
    fail "39b the copy is ${DUP_STATUS:-none}, want draft"
fi

DUP_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (SELECT status FROM events WHERE id = '${DUP_ID}')
         || '|' || (SELECT count(*) FROM orders WHERE event_id = '${DUP_ID}')
         || '|' || (SELECT coalesce(sum(quantity_sold), 0) FROM ticket_types
                      WHERE event_id = '${DUP_ID}')
         || '|' || (SELECT count(*) FROM check_in_records c JOIN tickets t ON t.id = c.ticket_id
                      WHERE t.event_id = '${DUP_ID}');" 2>&1 | tr -d '\r\n ')"
if [ "$DUP_ROW" = "draft|0|0|0" ]; then
    pass "40  PostgreSQL confirms the copy is draft with 0 orders, 0 sold, 0 check-ins"
else
    fail "40  the copy reports ${DUP_ROW:-nothing}, want draft|0|0|0"
fi

CONFIG_MATCH="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (o.venue_name IS NOT DISTINCT FROM d.venue_name
             AND o.timezone IS NOT DISTINCT FROM d.timezone
             AND o.capacity IS NOT DISTINCT FROM d.capacity
             AND d.duplicated_from_event_id = o.id)
       FROM events o, events d WHERE o.id = '${EVENT_ID}' AND d.id = '${DUP_ID}';" \
    2>&1 | tr -d '\r\n ')"
if [ "$CONFIG_MATCH" = "t" ]; then
    pass "40b the copy carries the same venue, timezone and capacity, and records its origin"
else
    fail "40b the copy's configuration does not match the original"
fi

TYPES_MATCH="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT (SELECT count(*) FROM ticket_types WHERE event_id = '${DUP_ID}')
         || '|' || (SELECT count(*) FROM ticket_types WHERE event_id = '${EVENT_ID}');" \
    2>&1 | tr -d '\r\n ')"
if [ -n "$TYPES_MATCH" ] && [ "${TYPES_MATCH%%|*}" = "${TYPES_MATCH##*|}" ]; then
    pass "40c every ticket type was copied (${TYPES_MATCH%%|*}), with the inventory reset to zero"
else
    fail "40c the copy has ${TYPES_MATCH%%|*} ticket types, the original has ${TYPES_MATCH##*|}"
fi

# A fresh draft has nothing to report.
request GET /api/v1/events/"$DUP_ID"/analytics "$TOKEN" -
DUP_SOLD="$(json_get "$HTTP_BODY" analytics.tickets_sold)"
DUP_REV="$(json_get "$HTTP_BODY" analytics.gross_revenue_kzt)"
if [ "$DUP_SOLD" = "0" ] && [ "$DUP_REV" = "0.00" ]; then
    pass "41  the copy's analytics start at 0 tickets and 0.00 KZT"
else
    fail "41  the copy reports ${DUP_SOLD:-none} sold / ${DUP_REV:-none} KZT, want 0 / 0.00"
fi

# The original keeps its sales.
request GET /api/v1/events/"$EVENT_ID"/analytics "$TOKEN" -
if [ "$(json_get "$HTTP_BODY" analytics.tickets_sold)" = "$DB_SOLD" ]; then
    pass "41b duplicating left the original's figures untouched (${DB_SOLD} sold)"
else
    fail "41b the original now reports $(json_get "$HTTP_BODY" analytics.tickets_sold) sold, want ${DB_SOLD}"
fi

# Drafts are listed under their own lifecycle filter.
request GET "/api/v1/events/mine?lifecycle=draft" "$TOKEN" -
expect_status 200 "42  the dashboard filters events by lifecycle"
if python3 -c '
import json, sys
ids = [e["id"] for e in json.loads(sys.argv[1]).get("events", [])]
sys.exit(0 if sys.argv[2] in ids else 1)
' "$HTTP_BODY" "$DUP_ID" 2>/dev/null; then
    pass "42b the new draft appears under the draft filter"
else
    fail "42b the new draft is missing from the draft filter"
fi

# --- Phase 10: refunds and notifications (SRS 4.9, 4.10) ---------------------
#
# The event is suspended by now, which does not stop a refund: money already
# taken can always be given back.

request GET /api/v1/events/"$EVENT_ID"/orders "$TOKEN" -
expect_status 200 "43  the organizer reads their attendee list"

REFUND_ORDER_ID="$(python3 -c '
import json, sys
try:
    orders = json.loads(sys.argv[1]).get("orders", [])
except Exception:
    orders = []
for o in orders:
    if o.get("refundable"):
        print(o["id"]); break
' "$HTTP_BODY")"

if [ -n "$REFUND_ORDER_ID" ]; then
    pass "43b at least one order is marked refundable"
    info "refunding order: $REFUND_ORDER_ID"
else
    fail "43b no refundable order in the attendee list"
fi

if [ -n "$REFUND_ORDER_ID" ]; then
    # The tickets work before the refund, which is what makes the "after"
    # meaningful rather than tickets that never worked.
    LIVE_BEFORE="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
        "SELECT count(*) FROM tickets WHERE order_id = '${REFUND_ORDER_ID}'
            AND status IN ('valid','checked_in');" 2>&1 | tr -d '\r\n ')"
    REFUND_QR="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
        "SELECT qr_token FROM tickets WHERE order_id = '${REFUND_ORDER_ID}' LIMIT 1;" \
        2>&1 | tr -d '\r\n ')"

    # An attendee cannot refund their own order.
    request POST /api/v1/orders/"$REFUND_ORDER_ID"/refund - -
    expect_status 401 "44  an anonymous caller cannot refund an order"

    request POST /api/v1/orders/"$REFUND_ORDER_ID"/refund "$BUYER_TOKEN" -
    expect_status 403 "44b the ticket buyer cannot refund their own order"

    request POST /api/v1/orders/"$REFUND_ORDER_ID"/refund "$TOKEN" \
        '{"reason":"Smoke-test refund"}'
    expect_status 200 "45  the organizer refunds the order"

    if [ "$(json_get "$HTTP_BODY" order.status)" = "refunded" ]; then
        pass "45b the order comes back refunded"
    else
        fail "45b order status is $(json_get "$HTTP_BODY" order.status), want refunded"
    fi
    if [ "$(json_get "$HTTP_BODY" voided_tickets)" = "$LIVE_BEFORE" ]; then
        pass "45c every live ticket on the order was voided (${LIVE_BEFORE})"
    else
        fail "45c voided $(json_get "$HTTP_BODY" voided_tickets) tickets, want ${LIVE_BEFORE}"
    fi

    REFUND_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
        "SELECT (SELECT o.status FROM orders o WHERE o.id = '${REFUND_ORDER_ID}')
             || '|' || (SELECT p.status FROM payments p WHERE p.order_id = '${REFUND_ORDER_ID}' LIMIT 1)
             || '|' || (SELECT r.status FROM refunds r WHERE r.order_id = '${REFUND_ORDER_ID}' LIMIT 1)
             || '|' || (SELECT count(*) FROM tickets t WHERE t.order_id = '${REFUND_ORDER_ID}'
                          AND t.status IN ('valid','checked_in'))
             || '|' || (SELECT count(*) FROM audit_logs a
                         WHERE a.entity_id = '${REFUND_ORDER_ID}' AND a.action = 'order.refunded');" \
        2>&1 | tr -d '\r\n ')"
    if [ "$REFUND_ROW" = "refunded|refunded|succeeded|0|1" ]; then
        pass "46  PostgreSQL: order and payment refunded, refund succeeded, 0 live tickets, 1 audit entry"
    else
        fail "46  the refund rows read ${REFUND_ROW:-nothing}, want refunded|refunded|succeeded|0|1"
    fi

    # SRS 4.9: refunded tickets become invalid. This is the Phase 6 gate.
    request POST /api/v1/events/"$EVENT_ID"/check-in "$TOKEN" \
        "{\"qr_token\":\"${REFUND_QR}\"}"
    expect_status 409 "47  the gate refuses a refunded ticket"
    if [ "$(json_get "$HTTP_BODY" error.code)" = "ticket_not_valid" ]; then
        pass "47b the scanner is told the ticket is not valid"
    else
        fail "47b scan refusal code is $(json_get "$HTTP_BODY" error.code), want ticket_not_valid"
    fi

    # Refunding twice is refused rather than writing a second refund.
    request POST /api/v1/orders/"$REFUND_ORDER_ID"/refund "$TOKEN" -
    expect_status 409 "48  a second refund on the same order is refused"
    if [ "$(json_get "$HTTP_BODY" error.code)" = "already_refunded" ]; then
        pass "48b the refusal is an explicit already_refunded"
    else
        fail "48b refusal code is $(json_get "$HTTP_BODY" error.code), want already_refunded"
    fi

    # SRS 4.10: the attendee is notified. The outbox row proves it was raised.
    NOTIFICATIONS="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
        "SELECT string_agg(type || ':' || status, ',' ORDER BY created_at)
           FROM notifications WHERE order_id = '${REFUND_ORDER_ID}';" 2>&1 | tr -d '\r\n ')"
    # Membership rather than an exact sequence: a support reply on this order
    # also notifies (Phase 12), and this check is about the money ones.
    if printf '%s' "$NOTIFICATIONS" | grep -q "order.confirmation:sent" &&
       printf '%s' "$NOTIFICATIONS" | grep -q "refund.completed:sent"; then
        pass "49  both notifications were sent: purchase confirmation and refund completion"
    else
        fail "49  notifications read ${NOTIFICATIONS:-nothing}"
    fi
fi

# --- Phase 12: account recovery, the admin portal, uploads, attendee search --
#
# SRS 2.1, 4.1, 4.2, 4.8, 4.10, 4.12.

# --- password reset (SRS 4.1) ------------------------------------------------
# The response must not reveal whether an address has an account, so both are
# compared rather than only checking the happy path.

request POST /api/v1/auth/password-reset/request - "{\"email\":\"${EMAIL}\"}"
expect_status 202 "50  a reset is requested for a real account"
KNOWN_REPLY="$HTTP_BODY"

request POST /api/v1/auth/password-reset/request - \
    '{"email":"definitely.not.registered@biletflow.test"}'
expect_status 202 "50b the same request for an unknown address"
if [ "$HTTP_BODY" = "$KNOWN_REPLY" ]; then
    pass "50c both answers are identical, so the form cannot enumerate accounts"
else
    fail "50c the two responses differ, which leaks who has an account"
fi

# The token never leaves the database in plaintext, so the test reads it the
# way a person reads the console: out of the notification body.
RESET_TOKEN="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT substring(body from 'reset-password\?token=([A-Za-z0-9_-]+)')
       FROM notifications
      WHERE type = 'account.password_reset' AND recipient_email = '${EMAIL}'
      ORDER BY created_at DESC LIMIT 1;" 2>&1 | tr -d '\r\n ')"

if [ -n "$RESET_TOKEN" ]; then
    pass "51  the reset email carries a token"
else
    fail "51  no reset token was emailed"
fi

STORED_PLAINTEXT="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT count(*) FROM user_tokens WHERE token_hash = '${RESET_TOKEN}';" 2>&1 | tr -d '\r\n ')"
if [ "$STORED_PLAINTEXT" = "0" ]; then
    pass "51b the database stores a hash, not the token itself"
else
    fail "51b the plaintext token is stored in user_tokens"
fi

if [ -n "$RESET_TOKEN" ]; then
    NEW_PASSWORD="a completely new passphrase"

    request POST /api/v1/auth/password-reset - \
        "{\"token\":\"${RESET_TOKEN}\",\"password\":\"${NEW_PASSWORD}\"}"
    expect_status 200 "52  the token sets a new password"

    request POST /api/v1/auth/login - "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}"
    expect_status 401 "52b the old password no longer works"

    request POST /api/v1/auth/login - "{\"email\":\"${EMAIL}\",\"password\":\"${NEW_PASSWORD}\"}"
    expect_status 200 "52c the new password does"
    # Everything after this point uses the refreshed session.
    TOKEN="$(json_get "$HTTP_BODY" access_token)"

    request POST /api/v1/auth/password-reset - \
        "{\"token\":\"${RESET_TOKEN}\",\"password\":\"trying the same code twice\"}"
    expect_status 400 "53  a reset code is good exactly once"
    if [ "$(json_get "$HTTP_BODY" error.code)" = "token_invalid" ]; then
        pass "53b the refusal is an explicit token_invalid"
    else
        fail "53b refusal code is $(json_get "$HTTP_BODY" error.code), want token_invalid"
    fi
fi

# --- email verification (SRS 4.1) --------------------------------------------

request POST /api/v1/auth/verify-email/request "$TOKEN" -
expect_status 202 "54  the account asks for a confirmation email"

VERIFY_TOKEN="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT substring(body from 'verify-email\?token=([A-Za-z0-9_-]+)')
       FROM notifications
      WHERE type = 'account.email_verification' AND recipient_email = '${EMAIL}'
      ORDER BY created_at DESC LIMIT 1;" 2>&1 | tr -d '\r\n ')"

if [ -n "$VERIFY_TOKEN" ]; then
    request POST /api/v1/auth/verify-email - "{\"token\":\"${VERIFY_TOKEN}\"}"
    expect_status 200 "54b the emailed code confirms the address"

    ACCOUNT_STATE="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
        "SELECT status || '/' || (email_verified_at IS NOT NULL)
           FROM users WHERE email = '${EMAIL}';" 2>&1 | tr -d '\r\n ')"
    if [ "$ACCOUNT_STATE" = "active/true" ]; then
        pass "54c PostgreSQL records the account as active and verified"
    else
        fail "54c the account reads ${ACCOUNT_STATE:-nothing}, want active/true"
    fi
else
    fail "54b no verification token was emailed"
fi

# --- image upload (SRS 4.2) --------------------------------------------------
# A 1x1 PNG, written as bytes so the sniffer sees a real image.

UPLOAD_FILE="$(mktemp -t biletflow_upload).png"
printf '\211PNG\r\n\032\n\0\0\0\015IHDR\0\0\0\001\0\0\0\001\010\006\0\0\0\037\025\304\211\0\0\0\012IDATx\234c\0\001\0\0\005\0\001\015\012-\264\0\0\0\0IEND\256B`\202' > "$UPLOAD_FILE"

UPLOAD_BODY="$(curl -sS -X POST "${API_URL}/api/v1/uploads/images" \
    -H "Authorization: Bearer ${TOKEN}" -F "file=@${UPLOAD_FILE}" 2>/dev/null)"
BANNER_URL="$(json_get "$UPLOAD_BODY" url)"

if [ -n "$BANNER_URL" ]; then
    pass "55  an organizer uploads an event banner"
    info "banner: $BANNER_URL"
else
    fail "55  the upload returned no URL: $UPLOAD_BODY"
fi

# A shell script wearing a .png name is not an image.
printf '#!/bin/sh\necho not an image\n' > "$UPLOAD_FILE"
UPLOAD_STATUS="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${API_URL}/api/v1/uploads/images" \
    -H "Authorization: Bearer ${TOKEN}" -F "file=@${UPLOAD_FILE}" 2>/dev/null)"
if [ "$UPLOAD_STATUS" = "415" ]; then
    pass "55b a non-image is refused whatever the filename says"
else
    fail "55b uploading a script returned ${UPLOAD_STATUS}, want 415"
fi
rm -f "$UPLOAD_FILE"

if [ -n "$BANNER_URL" ]; then
    SERVED="$(curl -sS -o /dev/null -w '%{http_code}' "$BANNER_URL" 2>/dev/null)"
    if [ "$SERVED" = "200" ]; then
        pass "56  the stored banner is served back, so the <img> resolves"
    else
        fail "56  fetching the banner returned ${SERVED}"
    fi

    # SRS 4.2 and 4.9: both fields reach the public page.
    request PATCH /api/v1/events/"$EVENT_ID" "$TOKEN" \
        "{\"cover_image_url\":\"${BANNER_URL}\",\"refund_policy\":\"Full refunds up to 7 days before the event.\"}"
    expect_status 200 "56b the event carries a banner and a refund policy"

    request GET /api/v1/public/events/"$SLUG" - -
    if [ "$(json_get "$HTTP_BODY" event.cover_image_url)" = "$BANNER_URL" ] &&
       [ "$(json_get "$HTTP_BODY" event.refund_policy)" = "Full refunds up to 7 days before the event." ]; then
        pass "56c the public page shows both"
    else
        fail "56c the public page is missing the banner or the policy"
    fi
fi

# --- manual attendee search (SRS 4.8) ----------------------------------------

request GET /api/v1/events/"$EVENT_ID"/attendees?q=Nurlan "$TOKEN" -
expect_status 200 "57  staff search the attendee list by name"

ATTENDEE_TICKET="$(python3 -c '
import json, sys
try:
    found = json.loads(sys.argv[1]).get("attendees", [])
except Exception:
    found = []
for a in found:
    if a.get("admissible"):
        print(a["ticket_id"]); break
' "$HTTP_BODY")"

if [ -n "$ATTENDEE_TICKET" ]; then
    pass "57b the search finds an admissible ticket"
else
    fail "57b no admissible ticket found by name"
fi

if echo "$HTTP_BODY" | grep -q "qr_token"; then
    fail "57c the attendee search leaked a QR token"
else
    pass "57c the results carry no QR token, so the search is not a skeleton key"
fi

request GET /api/v1/events/"$EVENT_ID"/attendees?q=Nurlan - -
expect_status 401 "58  an attendee list is not public"

if [ -n "$ATTENDEE_TICKET" ]; then
    request POST /api/v1/events/"$EVENT_ID"/check-in/manual "$TOKEN" \
        "{\"ticket_id\":\"${ATTENDEE_TICKET}\"}"
    expect_status 200 "59  staff check somebody in without a QR code"

    MANUAL_ROW="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
        "SELECT t.status || '/' || COALESCE(c.device_label, 'none')
           FROM tickets t
           LEFT JOIN check_in_records c ON c.ticket_id = t.id AND c.reversed_at IS NULL
          WHERE t.id = '${ATTENDEE_TICKET}';" 2>&1 | tr -d '\r\n ')"
    if [ "$MANUAL_ROW" = "checked_in/manualsearch" ]; then
        pass "59b PostgreSQL records the admission and how it was made"
    else
        fail "59b the ticket row reads ${MANUAL_ROW:-nothing}, want checked_in/manual search"
    fi

    # It is a real check-in, so the same duplicate protection applies.
    request POST /api/v1/events/"$EVENT_ID"/check-in/manual "$TOKEN" \
        "{\"ticket_id\":\"${ATTENDEE_TICKET}\"}"
    expect_status 409 "59c a second manual check-in is refused"
fi

# --- the administrative portal (SRS 2.1, 4.12) -------------------------------

# The smoke-test account was promoted to platform admin during the Phase 8
# checks, so the negative case needs an account that never was.
request POST /api/v1/auth/register - \
    "{\"email\":\"plain.${STAMP}@biletflow.test\",\"password\":\"an ordinary passphrase\",\"full_name\":\"Plain Organizer\"}"
PLAIN_TOKEN="$(json_get "$HTTP_BODY" access_token)"

request GET "/api/v1/admin/search?q=${EMAIL}" "$PLAIN_TOKEN" -
expect_status 403 "60  an ordinary account cannot reach the admin portal"

request GET "/api/v1/admin/search?q=${EMAIL}" - -
expect_status 401 "60a nor can an anonymous one"

request GET "/api/v1/admin/search?q=${EMAIL}" "$TOKEN" -
expect_status 200 "60b a platform admin can"

ADMIN_HITS="$(python3 -c '
import json, sys
try:
    r = json.loads(sys.argv[1])["results"]
except Exception:
    print("0/0/0"); sys.exit()
print("%d/%d/%d" % (len(r["users"]), len(r["events"]), len(r["orders"])))
' "$HTTP_BODY")"
if [ "${ADMIN_HITS%%/*}" = "1" ]; then
    pass "60c searching an email finds that user (users/events/orders: ${ADMIN_HITS})"
else
    fail "60c the search found ${ADMIN_HITS} users/events/orders, want 1 user"
fi

REPORT_FILE="$(mktemp -t biletflow_report).csv"
REPORT_HEADERS="$(curl -sS -D - -o "$REPORT_FILE" \
    "${API_URL}/api/v1/admin/reports/events.csv" \
    -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)"

if printf '%s' "$REPORT_HEADERS" | grep -qi "content-type: text/csv"; then
    pass "61  the operational report is served as CSV"
else
    fail "61  the report Content-Type is not text/csv"
fi
if printf '%s' "$REPORT_HEADERS" | grep -qi "content-disposition: attachment"; then
    pass "61b it downloads as an attachment"
else
    fail "61b the report is not sent as an attachment"
fi

REPORT_CHECK="$(python3 -c '
import csv, sys
with open(sys.argv[1], newline="") as handle:
    rows = list(csv.DictReader(handle))
wanted = sys.argv[2]
for row in rows:
    if row["event_id"] == wanted:
        print("found:" + row["tickets_sold"] + ":" + row["gross_kzt"]); break
else:
    print("missing")
' "$REPORT_FILE" "$EVENT_ID" 2>/dev/null)"

if [ "${REPORT_CHECK%%:*}" = "found" ]; then
    pass "61c the report parses as CSV and contains this event (${REPORT_CHECK})"
else
    fail "61c the smoke-test event is not in the report"
fi
rm -f "$REPORT_FILE"

# --- Phase 13: the requirements the earlier phases had left open -------------
#   SRS 4.9  cancelling a free registration
#   SRS 4.12 suspending a user, the report queue, configurable platform settings
#   SRS 4.1  organizer profile and self-service password change
#   SRS 4.13 assigning a support case
#   SRS 4.6  a payment that fails, issuing nothing
printf '\n%s\n' "${BOLD}Phase 13 - the remaining SRS requirements${OFF}"

# 62. A free registration is cancelled, not refunded.
request POST /api/v1/events "$TOKEN" \
    "{\"title\":\"Smoke Free Event\",\"starts_at\":\"2027-06-01T10:00:00Z\",
      \"ends_at\":\"2027-06-01T12:00:00Z\",\"timezone\":\"Asia/Almaty\",
      \"venue_name\":\"Hall\",\"venue_address\":\"Almaty\",\"capacity\":50}"
FREE_EVENT_ID="$(json_get "$HTTP_BODY" event.id)"
request POST /api/v1/events/"$FREE_EVENT_ID"/ticket-types "$TOKEN" \
    '{"name":"Free Entry","price_kzt":"0","quantity_total":10}'
FREE_TYPE_ID="$(json_get "$HTTP_BODY" ticket_type.id)"
request POST /api/v1/events/"$FREE_EVENT_ID"/publish "$TOKEN" -

request POST /api/v1/events/"$FREE_EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Free Attendee\",\"buyer_email\":\"smoke.free@biletflow.test\",
      \"items\":[{\"ticket_type_id\":\"${FREE_TYPE_ID}\",\"quantity\":1}]}"
expect_status 201 "62  a free registration is created"
FREE_ORDER_ID="$(json_get "$HTTP_BODY" order.id)"

request GET /api/v1/events/"$FREE_EVENT_ID"/orders "$TOKEN" -
FREE_REFUNDABLE="$(json_get "$HTTP_BODY" orders)"
if printf '%s' "$FREE_REFUNDABLE" | grep -q '"cancellable": *true' &&
   printf '%s' "$FREE_REFUNDABLE" | grep -q '"refundable": *false'; then
    pass "62a the order is flagged cancellable, not refundable"
else
    fail "62a a free order is mis-flagged; the dashboard would offer a button that fails"
    info "orders: $FREE_REFUNDABLE"
fi

request POST /api/v1/orders/"$FREE_ORDER_ID"/refund "$TOKEN" -
expect_status 409 "62b refunding a free registration is refused cleanly, not with a 500"

request POST /api/v1/orders/"$FREE_ORDER_ID"/cancel "$TOKEN" '{"reason":"smoke test"}'
expect_status 200 "62c the organizer cancels the free registration"

FREE_STATE="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT o.status || '|' || t.status || '|' || tt.quantity_sold
       FROM orders o
       JOIN tickets t ON t.order_id = o.id
       JOIN ticket_types tt ON tt.id = t.ticket_type_id
      WHERE o.id = '${FREE_ORDER_ID}';" 2>/dev/null | tr -d '[:space:]')"
if [ "$FREE_STATE" = "cancelled|cancelled|0" ]; then
    pass "62d PostgreSQL shows the order and ticket void and the place back on sale"
else
    fail "62d expected cancelled|cancelled|0, got '${FREE_STATE}'"
fi

request POST /api/v1/orders/"$FREE_ORDER_ID"/cancel "$TOKEN" -
expect_status 409 "62e a second cancellation is refused"

# 63. A declined payment issues nothing (SRS 4.6, 4.10).
# The Phase 8 event is deliberately left suspended, so the decline is exercised
# on a fresh paid tier that is genuinely open for business.
request POST /api/v1/events/"$FREE_EVENT_ID"/ticket-types "$TOKEN" \
    '{"name":"Paid Tier","price_kzt":"5000","quantity_total":10}'
DECLINE_TYPE_ID="$(json_get "$HTTP_BODY" ticket_type.id)"
request POST /api/v1/events/"$FREE_EVENT_ID"/activation "$TOKEN" \
    '{"confirm_identity":true,"confirm_payout":true,"accept_terms":true,"pay_activation_fee":true}'

request POST /api/v1/events/"$FREE_EVENT_ID"/checkout - \
    "{\"buyer_name\":\"Declined Buyer\",\"buyer_email\":\"smoke@decline.simulator.biletflow.kz\",
      \"items\":[{\"ticket_type_id\":\"${DECLINE_TYPE_ID}\",\"quantity\":1}]}"
expect_status 402 "63  the simulated provider declines the payment"
DECLINE_ROWS="$(docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAX -c \
    "SELECT count(*) FROM orders WHERE buyer_email = 'smoke@decline.simulator.biletflow.kz';" \
    2>/dev/null | tr -d '[:space:]')"
if [ "$DECLINE_ROWS" = "0" ]; then
    pass "63a a failed transaction created no order and no ticket"
else
    fail "63a a declined payment left ${DECLINE_ROWS} order(s) behind"
fi

# 64. The organizer profile carries contact and payout information (SRS 4.1).
request PATCH /api/v1/auth/profile "$TOKEN" \
    '{"display_name":"Smoke Promotions","contact_email":"hello@smoke.test",
      "website_url":"https://smoke.test"}'
expect_status 200 "64  the organizer fills in their profile"
request GET /api/v1/auth/profile "$TOKEN" -
if [ "$(json_get "$HTTP_BODY" profile.display_name)" = "Smoke Promotions" ]; then
    pass "64a it reads back on a fresh request"
else
    fail "64a the profile did not persist"
fi
if printf '%s' "$HTTP_BODY" | grep -q '"masked_account"' &&
   ! printf '%s' "$HTTP_BODY" | grep -q 'provider_account_ref'; then
    pass "64b the payout destination is masked and its provider reference is withheld"
else
    fail "64b the payout account is missing or leaks its provider reference"
    info "profile: $HTTP_BODY"
fi

# 65. A signed-in user changes their own password (SRS 4.1).
# Phase 12 already reset this account's password, so the current one is
# NEW_PASSWORD rather than the address's original.
CURRENT_PASSWORD="${NEW_PASSWORD:-$PASSWORD}"
request POST /api/v1/auth/password "$TOKEN" \
    "{\"current_password\":\"${CURRENT_PASSWORD}\",\"new_password\":\"a different passphrase\"}"
expect_status 204 "65  the password is changed"
request POST /api/v1/auth/login - \
    "{\"email\":\"${EMAIL}\",\"password\":\"${CURRENT_PASSWORD}\"}"
expect_status 401 "65a the old password stops working"
request POST /api/v1/auth/login - \
    "{\"email\":\"${EMAIL}\",\"password\":\"a different passphrase\"}"
expect_status 200 "65b the new one works"
TOKEN="$(json_get "$HTTP_BODY" access_token)"
request POST /api/v1/auth/password "$TOKEN" \
    '{"current_password":"not it","new_password":"another passphrase"}'
expect_status 403 "65c a change without the current password is refused"

# 66. An attendee reports an event and an admin reviews it (SRS 4.12).
request POST /api/v1/auth/register - \
    "{\"email\":\"reporter.${STAMP}@biletflow.test\",\"password\":\"correct horse battery\",
      \"full_name\":\"Smoke Reporter\"}"
REPORTER_TOKEN="$(json_get "$HTTP_BODY" access_token)"
REPORTER_ID="$(json_get "$HTTP_BODY" user.id)"

request POST /api/v1/events/"$FREE_EVENT_ID"/report "$REPORTER_TOKEN" \
    '{"reason":"misleading","details":"The description does not match the venue."}'
expect_status 201 "66  an attendee reports an event"
REPORT_ID="$(json_get "$HTTP_BODY" report.id)"

request POST /api/v1/events/"$FREE_EVENT_ID"/report "$REPORTER_TOKEN" '{"reason":"spam"}'
expect_status 409 "66a a second report from the same person is one complaint, not two"

request GET /api/v1/admin/event-reports?status=open "$REPORTER_TOKEN" -
expect_status 403 "66b the moderation queue is not readable by an ordinary account"

request GET /api/v1/admin/event-reports?status=open "$TOKEN" -
expect_status 200 "66c a platform admin reads the queue"

request PATCH /api/v1/admin/event-reports/"$REPORT_ID" "$TOKEN" \
    '{"status":"dismissed","resolution":"Checked; the listing is accurate."}'
expect_status 200 "66d the admin records a decision"
if [ -n "$(json_get "$HTTP_BODY" report.reviewed_at)" ]; then
    pass "66e the decision names its reviewer and when it was made"
else
    fail "66e a decided report has no reviewer recorded"
fi

# 67. Suspending a user stops their events selling (SRS 4.12).
request POST /api/v1/admin/users/"$REPORTER_ID"/suspend "$TOKEN" \
    '{"reason":"smoke test"}'
expect_status 200 "67  a platform admin suspends a user"
request GET /api/v1/auth/me "$REPORTER_TOKEN" -
expect_status 403 "67a their existing token stops working on the very next request"
request POST /api/v1/admin/users/"$REPORTER_ID"/unsuspend "$TOKEN" -
expect_status 200 "67b the suspension is lifted"

# 68. The activation fee is configurable rather than compiled in (SRS 4.12).
request GET /api/v1/admin/settings "$TOKEN" -
expect_status 200 "68  the admin reads the platform settings"
request PATCH /api/v1/admin/settings/activation_fee_kzt "$TOKEN" '{"value":"7500.00"}'
expect_status 200 "68a the activation fee is changed without a rebuild"
request PATCH /api/v1/admin/settings/activation_fee_kzt "$TOKEN" '{"value":"free"}'
expect_status 422 "68b a fee that is not an amount is refused"
request PATCH /api/v1/admin/settings/activation_fee_kzt "$TOKEN" '{"value":"5000.00"}'
expect_status 200 "68c the fee is restored"

# 69. A support case is assigned to a named person (SRS 4.13).
if [ -n "${CASE_ID:-}" ]; then
    request POST /api/v1/support/cases/"$CASE_ID"/assign "$TOKEN" \
        "{\"email\":\"${EMAIL}\"}"
    expect_status 200 "69  the organizer assigns the case to a named person"
    if [ -n "$(json_get "$HTTP_BODY" case.assigned_to_user_id)" ]; then
        pass "69a the case records who is handling it"
    else
        fail "69a the case is not assigned"
    fi
    request POST /api/v1/support/cases/"$CASE_ID"/assign "$BUYER_TOKEN" \
        "{\"email\":\"${EMAIL}\"}"
    expect_status 403 "69b an attendee cannot decide who handles their case"
else
    info "69  skipped - no support case id from the earlier phase"
fi

# --- clean up ----------------------------------------------------------------
request POST /api/v1/events/"$EVENT_ID"/cancel "$TOKEN" -
docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -qtAX \
    -c "DELETE FROM user_tokens WHERE user_id = '${USER_ID}';
        DELETE FROM notifications WHERE event_id = '${EVENT_ID}';
        DELETE FROM notifications WHERE recipient_email = '${EMAIL}';
        DELETE FROM refunds WHERE order_id IN (SELECT id FROM orders WHERE event_id = '${EVENT_ID}');
        DELETE FROM paid_sales_activations WHERE event_id = '${EVENT_ID}';
        DELETE FROM payments WHERE event_id = '${EVENT_ID}';
        DELETE FROM support_messages WHERE support_case_id IN (SELECT id FROM support_cases WHERE event_id = '${EVENT_ID}');
        DELETE FROM support_cases WHERE event_id = '${EVENT_ID}';
        DELETE FROM promo_redemptions WHERE campaign_id IN (SELECT id FROM campaigns WHERE event_id = '${EVENT_ID}');
        DELETE FROM tickets WHERE event_id = '${EVENT_ID}';
        DELETE FROM payments WHERE order_id IN (SELECT id FROM orders WHERE event_id = '${EVENT_ID}');
        DELETE FROM orders WHERE event_id = '${EVENT_ID}';
        DELETE FROM ticket_types WHERE event_id = '${EVENT_ID}';
        DELETE FROM ticket_types WHERE event_id = '${DUP_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM events WHERE id = '${DUP_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM events WHERE id = '${EVENT_ID}';" >/dev/null 2>&1
docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -qtAX \
    -c "DELETE FROM event_reports WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM notifications WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM notifications WHERE recipient_email IN ('smoke.free@biletflow.test',
                                                            'smoke@decline.simulator.biletflow.kz');
        DELETE FROM payments WHERE order_id IN (SELECT id FROM orders WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}');
        DELETE FROM tickets WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM orders WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM paid_sales_activations WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM payments WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM ticket_types WHERE event_id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM events WHERE id = '${FREE_EVENT_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM payout_accounts WHERE organizer_profile_id IN
            (SELECT id FROM organizer_profiles WHERE user_id = '${USER_ID}');
        DELETE FROM organizer_profiles WHERE user_id = '${USER_ID}';
        DELETE FROM users WHERE id = '${REPORTER_ID:-00000000-0000-0000-0000-000000000000}';
        DELETE FROM users WHERE id = '${USER_ID}';" >/dev/null 2>&1
info "cleaned up the smoke-test user and event"

printf '%s\n' "-----------------------------------------------------------"
if [ "$failures" -eq 0 ]; then
    printf '%s\n' "${GREEN}${BOLD}All acceptance checks passed.${OFF}"
    exit 0
fi
printf '%s\n' "${RED}${BOLD}${failures} check(s) failed.${OFF}"
exit 1
