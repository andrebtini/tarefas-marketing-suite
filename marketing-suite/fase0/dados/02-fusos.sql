-- 02-fusos.sql | spec secao 3, F0-034 | idempotente
BEGIN;
UPDATE workspaces SET timezone = 'America/Sao_Paulo'
 WHERE slug = 'marketing-suite' AND timezone <> 'America/Sao_Paulo';
UPDATE projects SET timezone = 'America/Sao_Paulo'
 WHERE workspace_id = (SELECT id FROM workspaces WHERE slug = 'marketing-suite')
   AND deleted_at IS NULL AND timezone <> 'America/Sao_Paulo';
COMMIT;
SELECT 'espaco', slug, timezone FROM workspaces WHERE slug = 'marketing-suite';
SELECT 'projeto', identifier, timezone FROM projects
 WHERE workspace_id = (SELECT id FROM workspaces WHERE slug = 'marketing-suite') AND deleted_at IS NULL;
