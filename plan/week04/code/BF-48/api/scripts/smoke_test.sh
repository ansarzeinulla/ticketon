#!/usr/bin/env bash
#
# BiletFlow API acceptance checks, over real HTTP with cURL.
#
# The same walk-through you would do by hand in Postman:
#   accounts: register -> login -> the token opens a protected route
#   events:   create, publish, read publicly
#   selling:  add a ticket type -> paid sales are refused until the
#             activation checklist is done -> buy through the simulated
#             checkout -> confirm the inventory moved -> confirm overselling
#             is refused
#   tickets:  download the A4 PDF ticket and its QR preview, export the event
#             as an .ics file
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

printf '%s\n' "${BOLD}BiletFlow API acceptance checks${OFF}  ->  $API_URL"
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

if [ "$ORDER_STATUS" = "paid" ] && [ "$ORDER_TOTAL" = "10000.00" ]; then
    pass "10b the order is paid for 10000.00 KZT"
    info "order id: $ORDER_ID"
else
    fail "10b order is ${ORDER_STATUS:-none} for ${ORDER_TOTAL:-none}, want paid / 10000.00"
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


# --- clean up ------------------------------------------------------------------
# audit_logs is append-only by trigger, so its rows stay; everything else the
# run created is removed so the script can be run again and again.
docker compose exec -T db psql -U "$DB_USER" -d "$DB_NAME" -qtAX \
    -c "DELETE FROM notifications WHERE event_id = '${EVENT_ID}';
        DELETE FROM notifications WHERE user_id = '${USER_ID}';
        DELETE FROM payments WHERE order_id IN (SELECT id FROM orders WHERE event_id = '${EVENT_ID}');
        DELETE FROM tickets WHERE event_id = '${EVENT_ID}';
        DELETE FROM orders WHERE event_id = '${EVENT_ID}';
        DELETE FROM paid_sales_activations WHERE event_id = '${EVENT_ID}';
        DELETE FROM payments WHERE event_id = '${EVENT_ID}';
        DELETE FROM ticket_types WHERE event_id = '${EVENT_ID}';
        DELETE FROM events WHERE id = '${EVENT_ID}';
        DELETE FROM payout_accounts WHERE organizer_profile_id IN
            (SELECT id FROM organizer_profiles WHERE user_id = '${USER_ID}');
        DELETE FROM organizer_profiles WHERE user_id = '${USER_ID}';
        DELETE FROM users WHERE id = '${USER_ID}';" >/dev/null 2>&1
info "cleaned up the smoke-test user and event"

printf '%s\n' "-----------------------------------------------------------"
if [ "$failures" -eq 0 ]; then
    printf '%s\n' "${GREEN}${BOLD}All acceptance checks passed.${OFF}"
    exit 0
fi
printf '%s\n' "${RED}${BOLD}${failures} check(s) failed.${OFF}"
exit 1
