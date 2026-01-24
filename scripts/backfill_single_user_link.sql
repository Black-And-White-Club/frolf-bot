-- backfill_single_user_link.sql
-- Purpose: insert a missing guild_membership for a known global user and discord id.
-- This is a targeted, single-row backfill. Replace placeholders before running.
-- Do NOT run this without a recent DB backup.

-- Placeholders to replace:
-- <GUILD_ID>   - the guild (server) id as text/uuid used in your DB
-- <DISCORD_ID> - the discord user's snowflake (numeric string)
-- <NORMALIZED_UDISC_USERNAME> - normalized udisc username to match the users row

BEGIN;

-- Verify user exists
SELECT id, user_id, udisc_username, udisc_name FROM users WHERE normalized_udisc_username = '<NORMALIZED_UDISC_USERNAME>' LIMIT 1;

-- Insert guild_membership if it does not already exist
INSERT INTO guild_memberships (user_id, guild_id, discord_id, role, joined_at)
SELECT u.user_id, '<GUILD_ID>'::text, '<DISCORD_ID>'::bigint, 'User', NOW()
FROM users u
WHERE u.normalized_udisc_username = '<NORMALIZED_UDISC_USERNAME>'
  AND NOT EXISTS (
    SELECT 1 FROM guild_memberships gm WHERE gm.guild_id = '<GUILD_ID>'::text AND gm.discord_id = '<DISCORD_ID>'::bigint
  );

-- Verify insertion
SELECT * FROM guild_memberships WHERE guild_id = '<GUILD_ID>'::text AND discord_id = '<DISCORD_ID>'::bigint;

COMMIT;
