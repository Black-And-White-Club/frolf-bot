-- backfill_rounds_teams.sql
-- Purpose: recompute `rounds.teams` JSONB from `rounds.participants` for rounds
-- that contain participants with team_id values. This script is for operator use
-- in a maintenance window. Run the dry-run section first and inspect results.

-- DRY-RUN: list up to 100 candidate rounds and the computed teams JSON
-- (do NOT run the UPDATE below until you have reviewed the results)

SELECT id,
(
  SELECT jsonb_agg(team) FROM (
    SELECT p ->> 'team_id' AS team_id,
      jsonb_build_object(
        'id', p ->> 'team_id',
        'members', jsonb_agg(jsonb_build_object(
            'user_id', NULLIF(p ->> 'user_id',''),
            'raw_name', p ->> 'raw_name'
        ))
      ) AS team
    FROM jsonb_array_elements(participants) AS p
    WHERE (p ->> 'team_id') IS NOT NULL AND (p ->> 'team_id') <> ''
    GROUP BY p ->> 'team_id'
  ) t
) AS computed_teams
FROM rounds
WHERE jsonb_path_exists(participants, '$[*] ? (@.team_id != null)')
LIMIT 100;

-- Example single-round update (replace <round_id> with actual uuid):
-- BEGIN;
-- WITH computed AS (
--   SELECT jsonb_agg(team) AS teams FROM (
--     SELECT p ->> 'team_id' AS team_id,
--       jsonb_build_object(
--         'id', p ->> 'team_id',
--         'members', jsonb_agg(jsonb_build_object('user_id', NULLIF(p ->> 'user_id',''), 'raw_name', p ->> 'raw_name'))
--       ) AS team
--     FROM jsonb_array_elements(
--       (SELECT participants FROM rounds WHERE id = '<round_id>')
--     ) AS p
--     WHERE (p ->> 'team_id') IS NOT NULL AND (p ->> 'team_id') <> ''
--     GROUP BY p ->> 'team_id'
--   ) t
-- )
-- UPDATE rounds SET teams = computed.teams FROM computed WHERE id = '<round_id>';
-- COMMIT;

-- BATCHED UPDATE pattern (recommended for large tables):
-- 1) Use the DRY-RUN select to collect candidate ids and verify.
-- 2) For each id (in batches), run the single-round update shown above.
-- Note: Avoid a single massive UPDATE to reduce lock contention.

-- After running updates, verify:
-- SELECT count(*) FROM rounds WHERE teams IS NULL;

-- End of script
