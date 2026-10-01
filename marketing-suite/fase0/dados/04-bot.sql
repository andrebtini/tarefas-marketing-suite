-- 04-bot.sql | spec secao 3, F0-036 (PROPOSTA F0-P5) | idempotente
BEGIN;
UPDATE users
   SET first_name = 'Marketing Suite', last_name = '', display_name = 'Marketing Suite'
 WHERE is_bot = true AND bot_type = 'WORKSPACE_SEED'
   AND username = 'bot_user_' || (SELECT id::text FROM workspaces WHERE slug = 'marketing-suite')
   AND display_name = 'Plane';
COMMIT;
SELECT username, first_name, display_name FROM users WHERE is_bot = true;
