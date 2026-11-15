-- mask-replica.sql — anonymize a restored copy of the production database.
--
-- Run ONLY against a local staging copy, never against production. It is
-- driven by scripts/replicate-db.sh, which restores the dump into a staging
-- database, runs this file, and only then renames the staging database to the
-- name the app connects to. Masking runs in one transaction and ends with
-- verification checks that abort (and roll back) if anything sensitive is left.
--
-- psql variables (all set by replicate-db.sh):
--   dev_password  password every masked account gets
--   scrub_text    on|off  replace free text outright instead of redacting PII in it
--   strip_media   on|off  null out image/document bytes stored in Postgres
--
-- When you add a model or a column that holds personal data, add it here too.

\set ON_ERROR_STOP on

-- Refuse to run anywhere that looks like an Azure-hosted database.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_settings WHERE name LIKE 'azure.%') THEN
    RAISE EXCEPTION 'mask-replica.sql refuses to run on an Azure server — run it on a local copy only';
  END IF;
END $$;

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- 1. Name map: real "First Last" → fake "First Last", built BEFORE users are
--    masked so volunteer names mentioned in free text can be replaced with the
--    same fake name the user record gets. Only full names are mapped; bare
--    first names are left alone because they collide with animal names
--    ("Max", "Bella"…).
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE fake_first(i int PRIMARY KEY, name text) ON COMMIT DROP;
INSERT INTO fake_first
SELECT ord - 1, n FROM unnest(ARRAY[
  'Alex','Blair','Casey','Dana','Emery','Finley','Gray','Harper','Indy','Jamie',
  'Kai','Logan','Morgan','Noel','Oakley','Parker','Quinn','Riley','Sage','Taylor',
  'Avery','Brook','Cameron','Drew','Ellis','Frankie','Hayden','Jesse','Kendall','Lane'
]) WITH ORDINALITY AS t(n, ord);

CREATE TEMP TABLE fake_last(i int PRIMARY KEY, name text) ON COMMIT DROP;
INSERT INTO fake_last
SELECT ord - 1, n FROM unnest(ARRAY[
  'Abbott','Bishop','Carver','Dalton','Ellison','Fletcher','Garner','Hollis','Irwin','Jensen',
  'Keller','Lowry','Mercer','Nolan','Osborne','Pruitt','Quincy','Ramsey','Sutton','Thorne',
  'Upton','Vance','Whitaker','Yates','Zeller','Barlow','Conway','Doyle','Easton','Frost'
]) WITH ORDINALITY AS t(n, ord);

CREATE TEMP TABLE user_mask ON COMMIT DROP AS
SELECT u.id,
       trim(u.first_name) AS real_first,
       trim(u.last_name)  AS real_last,
       f.name AS fake_first,
       l.name AS fake_last
FROM users u
JOIN fake_first f ON f.i = u.id % 30
JOIN fake_last  l ON l.i = (u.id * 7 + u.id / 30) % 30;

CREATE TEMP TABLE name_map ON COMMIT DROP AS
SELECT DISTINCT ON (lower(real_first || ' ' || real_last))
       -- escape regex metacharacters so names are matched literally
       regexp_replace(real_first || ' ' || real_last, '([.^$*+?()\[\]{}|\\-])', '\\\1', 'g') AS pattern,
       real_first || ' ' || real_last AS real_full,
       fake_first || ' ' || fake_last AS fake_full
FROM user_mask
WHERE length(real_first) >= 2 AND length(real_last) >= 2
ORDER BY lower(real_first || ' ' || real_last), id;

-- Redact PII embedded in free text: email addresses, US-style phone numbers,
-- and volunteers' full names.
CREATE FUNCTION pg_temp.mask_text(t text) RETURNS text
LANGUAGE plpgsql AS $fn$
DECLARE
  r record;
BEGIN
  IF t IS NULL OR t = '' THEN
    RETURN t;
  END IF;
  t := regexp_replace(t, '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}', '[email]', 'g');
  -- 555-123-4567, 555.123.4567, 555 123 4567, (555) 123-4567, +1 555-123-4567.
  -- Needs separators so microchip / ID numbers are not mistaken for phones.
  t := regexp_replace(t, '(\+?1[-. ]?)?\(?\m\d{3}\)?[-. ]\s?\d{3}[-. ]\d{4}\M', '[phone]', 'g');
  t := regexp_replace(t, '(\+?1[-. ]?)?\(\d{3}\)\s?\d{3}[-. ]?\d{4}\M', '[phone]', 'g');
  FOR r IN SELECT pattern, real_full, fake_full FROM name_map LOOP
    IF position(lower(r.real_full) IN lower(t)) > 0 THEN
      t := regexp_replace(t, '\m' || r.pattern || '\M', r.fake_full, 'gi');
    END IF;
  END LOOP;
  RETURN t;
END
$fn$;

CREATE FUNCTION pg_temp.mask_json(j jsonb) RETURNS jsonb
LANGUAGE sql AS $fn$
  -- Replacements contain no quotes or backslashes, so the text stays valid JSON.
  SELECT CASE WHEN j IS NULL THEN NULL ELSE pg_temp.mask_text(j::text)::jsonb END
$fn$;

-- ---------------------------------------------------------------------------
-- 2. Users and credentials
-- ---------------------------------------------------------------------------
-- One bcrypt hash (pgcrypto emits $2a$, which golang.org/x/crypto/bcrypt accepts).
CREATE TEMP TABLE dev_hash ON COMMIT DROP AS
SELECT crypt(:'dev_password', gen_salt('bf', 10)) AS hash;

UPDATE users u SET
  username                    = 'user' || u.id,
  first_name                  = m.fake_first,
  last_name                   = m.fake_last,
  email                       = 'user' || u.id || '@example.com',
  phone_number                = CASE WHEN u.phone_number <> ''
                                     THEN '555-01' || lpad((u.id % 100)::text, 2, '0')
                                     ELSE '' END,
  password                    = (SELECT hash FROM dev_hash),
  reset_token                 = '',
  reset_token_lookup          = '',
  reset_token_expiry          = NULL,
  setup_token                 = '',
  setup_token_lookup          = '',
  setup_token_expiry          = NULL,
  requires_password_setup     = false,
  failed_login_attempts       = 0,
  locked_until                = NULL,
  last_login                  = NULL,
  email_notifications_enabled = false
FROM user_mask m
WHERE m.id = u.id;

-- API tokens are live credentials for the real site. Drop them all.
DELETE FROM api_tokens;

-- ---------------------------------------------------------------------------
-- 3. Outbound integrations — a local copy must never post to the real GroupMe
--    chats or send coverage digests.
-- ---------------------------------------------------------------------------
UPDATE groups SET groupme_bot_id = '', groupme_enabled = false;
UPDATE shift_coverage_requests SET notified_at = COALESCE(notified_at, now());

-- ---------------------------------------------------------------------------
-- 4. Free text. Bite-incident details can name members of the public, so they
--    are always replaced. Everything else has PII redacted, or is replaced
--    outright with scrub_text=on.
-- ---------------------------------------------------------------------------
UPDATE animals SET quarantine_incident_details = '[redacted incident details]'
WHERE quarantine_incident_details <> '';
UPDATE animal_bq_incidents SET incident_details = '[redacted incident details]'
WHERE incident_details <> '';

\if :scrub_text
UPDATE animal_comments SET
  content  = 'Session note #' || id,
  metadata = CASE WHEN metadata IS NULL THEN NULL
                  ELSE metadata - 'session_goal' - 'session_outcome' - 'behavior_notes'
                                - 'medical_notes' - 'other_notes' END;
UPDATE comment_histories SET
  content  = 'Earlier version of session note #' || comment_id,
  metadata = CASE WHEN metadata IS NULL THEN NULL
                  ELSE metadata - 'session_goal' - 'session_outcome' - 'behavior_notes'
                                - 'medical_notes' - 'other_notes' END;
UPDATE updates       SET title = 'Update #' || id,       content = 'Update body #' || id;
UPDATE announcements SET title = 'Announcement #' || id, content = 'Announcement body #' || id;
UPDATE animals SET
  description   = CASE WHEN description   <> '' THEN 'Description for animal #' || id ELSE '' END,
  trainer_notes = CASE WHEN trainer_notes <> '' THEN 'Trainer notes for animal #' || id ELSE '' END;
UPDATE animal_images SET caption = '' WHERE caption <> '';
UPDATE animal_videos SET caption = '' WHERE caption <> '';
\else
UPDATE animal_comments   SET content = pg_temp.mask_text(content), metadata = pg_temp.mask_json(metadata);
UPDATE comment_histories SET content = pg_temp.mask_text(content), metadata = pg_temp.mask_json(metadata);
UPDATE updates           SET title = pg_temp.mask_text(title), content = pg_temp.mask_text(content);
UPDATE announcements     SET title = pg_temp.mask_text(title), content = pg_temp.mask_text(content);
UPDATE animals SET description   = pg_temp.mask_text(description),
                   trainer_notes = pg_temp.mask_text(trainer_notes);
UPDATE animal_images SET caption = pg_temp.mask_text(caption) WHERE caption <> '';
UPDATE animal_videos SET caption = pg_temp.mask_text(caption) WHERE caption <> '';
\endif

-- Care docs and settings are shelter content, but may carry staff contact info.
UPDATE protocols       SET content     = pg_temp.mask_text(content);
UPDATE scripts         SET description = pg_temp.mask_text(description);
UPDATE group_documents SET description = pg_temp.mask_text(description);
UPDATE site_settings   SET value       = pg_temp.mask_text(value);

-- Embeddings were computed from the unmasked text; drop them so they can't
-- leak it. With VOYAGE_API_KEY set locally, the reconciliation sweep refills
-- them from the masked text; without it search is keyword-only.
DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['animals', 'animal_comments', 'updates'] LOOP
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = 'public' AND table_name = t AND column_name = 'embedding') THEN
      EXECUTE format('UPDATE %I SET embedding = NULL WHERE embedding IS NOT NULL', t);
    END IF;
  END LOOP;
END $$;

-- ---------------------------------------------------------------------------
-- 5. Media stored in Postgres (photos can show volunteers' faces). Azure-stored
--    media is never copied — only its blob identifiers, which won't resolve.
-- ---------------------------------------------------------------------------
\if :strip_media
UPDATE animal_images   SET image_data = NULL             WHERE image_data IS NOT NULL;
UPDATE animals         SET protocol_document_data = NULL WHERE protocol_document_data IS NOT NULL;
UPDATE scripts         SET file_data = NULL              WHERE file_data IS NOT NULL;
UPDATE group_documents SET file_data = NULL              WHERE file_data IS NOT NULL;
\endif

-- ---------------------------------------------------------------------------
-- 6. Verify. Any failure raises, which rolls back the whole transaction and
--    makes replicate-db.sh discard the staging database.
-- ---------------------------------------------------------------------------
DO $$
DECLARE
  n bigint;
  email_re constant text := '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}';
BEGIN
  SELECT count(*) INTO n FROM users WHERE email NOT LIKE '%@example.com';
  IF n > 0 THEN RAISE EXCEPTION 'verify: % user emails not masked', n; END IF;

  SELECT count(*) INTO n FROM users
  WHERE reset_token <> '' OR setup_token <> '' OR reset_token_lookup <> '' OR setup_token_lookup <> '';
  IF n > 0 THEN RAISE EXCEPTION 'verify: % users still hold reset/setup tokens', n; END IF;

  SELECT count(*) INTO n FROM api_tokens;
  IF n > 0 THEN RAISE EXCEPTION 'verify: % API tokens remain', n; END IF;

  SELECT count(*) INTO n FROM groups WHERE groupme_bot_id <> '' OR groupme_enabled;
  IF n > 0 THEN RAISE EXCEPTION 'verify: % groups still have GroupMe configured', n; END IF;

  SELECT count(*) INTO n FROM animal_comments WHERE content ~ email_re OR metadata::text ~ email_re;
  IF n > 0 THEN RAISE EXCEPTION 'verify: % comments still contain an email address', n; END IF;

  SELECT count(*) INTO n FROM updates WHERE content ~ email_re OR title ~ email_re;
  IF n > 0 THEN RAISE EXCEPTION 'verify: % updates still contain an email address', n; END IF;

  SELECT count(*) INTO n FROM announcements WHERE content ~ email_re OR title ~ email_re;
  IF n > 0 THEN RAISE EXCEPTION 'verify: % announcements still contain an email address', n; END IF;

  SELECT count(*) INTO n FROM animal_bq_incidents
  WHERE incident_details NOT IN ('', '[redacted incident details]');
  IF n > 0 THEN RAISE EXCEPTION 'verify: % bite incidents not redacted', n; END IF;

  RAISE NOTICE 'masking verified';
END $$;

COMMIT;
