-- =============================================================================
-- BiletFlow - 01_extensions.sql
-- Extensions required by the schema. Safe to re-run.
-- =============================================================================

-- citext: case-insensitive email addresses and promo codes (SRS 4.1, 4.14).
CREATE EXTENSION IF NOT EXISTS citext;

-- pgcrypto: gen_random_uuid() is built into PG13+, but the extension also gives
-- digest()/hmac() which the QR-token signing work in later phases will need.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- btree_gist: required for the EXCLUDE constraint that prevents two concurrent
-- orders from holding the same seat (SRS 4.3.1, NFR "atomic reservation").
CREATE EXTENSION IF NOT EXISTS btree_gist;
