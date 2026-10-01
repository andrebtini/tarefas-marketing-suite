-- 01-perfis.sql | spec secao 3, F0-032 | idempotente: roda de novo a cada pessoa nova
BEGIN;
UPDATE profiles AS p
   SET language = 'pt-BR', start_of_the_week = 1
  FROM users AS u
 WHERE p.user_id = u.id AND u.is_bot = false
   AND (p.language <> 'pt-BR' OR p.start_of_the_week <> 1);
UPDATE users
   SET user_timezone = 'America/Sao_Paulo'
 WHERE is_bot = false AND user_timezone <> 'America/Sao_Paulo';
COMMIT;
SELECT u.email, p.language, p.start_of_the_week, u.user_timezone
  FROM profiles AS p JOIN users AS u ON u.id = p.user_id
 WHERE u.is_bot = false ORDER BY u.email;
