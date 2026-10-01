-- 01b-tema.sql | spec secao 3, F0-033 (PROPOSTA F0-P3) | idempotente
BEGIN;
UPDATE profiles AS p
   SET theme = jsonb_set(p.theme, '{theme}', to_jsonb(CASE p.theme->>'theme'
                 WHEN 'custom' THEN 'system'
                 WHEN 'light-contrast' THEN 'light'
                 WHEN 'dark-contrast' THEN 'dark' END))
  FROM users AS u
 WHERE p.user_id = u.id AND u.is_bot = false
   AND p.theme->>'theme' IN ('custom', 'light-contrast', 'dark-contrast');
COMMIT;
SELECT u.email, p.theme->>'theme' AS tema
  FROM profiles AS p JOIN users AS u ON u.id = p.user_id WHERE u.is_bot = false ORDER BY u.email;
