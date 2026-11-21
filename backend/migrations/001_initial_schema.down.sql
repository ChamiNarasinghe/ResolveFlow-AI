
-- ResolveFlow AI: Rollback initial database schema

-- Drop tables in reverse dependency order.

DROP TABLE IF EXISTS ai_predictions;
DROP TABLE IF EXISTS ticket_history;
DROP TABLE IF EXISTS ticket_comments;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
