#!/bin/bash
set -e

# Default UUIDs for local dev consistent with user logs
GUILD_UUID="579d45b8-1790-4e23-92ca-628caec9d263"
USER_UUID="410f3bd7-8caf-4d34-919e-208f8cfbee25"
DISCORD_GUILD_ID="1458214881651064955"
DISCORD_USER_ID="153320995397173249"

echo "Seeding local development data..."

docker-compose exec -T db psql -U user -d frolf_bot <<EOF
-- 1. Seed Guild Config
INSERT INTO guild_configs (uuid, guild_id, created_at, updated_at, is_active, signup_emoji, auto_setup_completed)
VALUES ('$GUILD_UUID', '$DISCORD_GUILD_ID', NOW(), NOW(), true, '🥏', true)
ON CONFLICT (guild_id) DO NOTHING;

-- 2. Seed Club (Crucial fix for FK error)
INSERT INTO clubs (uuid, name, discord_guild_id, created_at, updated_at)
VALUES ('$GUILD_UUID', 'Demo Server', '$DISCORD_GUILD_ID', NOW(), NOW())
ON CONFLICT (uuid) DO NOTHING;

-- 3. Seed User
INSERT INTO users (uuid, user_id, udisc_username, udisc_name, display_name, created_at, updated_at)
VALUES ('$USER_UUID', '$DISCORD_USER_ID', 'jacediscgolf', 'Jace', 'Jace', NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING;

-- 4. Seed Guild Membership
INSERT INTO guild_memberships (user_id, guild_id, discord_id, role, joined_at)
VALUES ('$DISCORD_USER_ID', '$DISCORD_GUILD_ID', $DISCORD_USER_ID, 'User', NOW())
ON CONFLICT (user_id, guild_id) DO NOTHING;

-- 5. Seed Club Membership
INSERT INTO club_memberships (user_uuid, club_uuid, role, joined_at, updated_at, display_name, source, external_id)
VALUES ('$USER_UUID', '$GUILD_UUID', 'player', NOW(), NOW(), 'Jace', 'discord', '$DISCORD_USER_ID')
ON CONFLICT (user_uuid, club_uuid) DO NOTHING;

EOF

echo "Done! Local data seeded."
