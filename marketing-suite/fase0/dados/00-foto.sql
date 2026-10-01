-- 00-foto.sql | spec secao 3, F0-031 | so leitura
\pset format unaligned
\pset fieldsep ' | '
\pset footer off
SELECT 'perfil', u.email, p.language, p.start_of_the_week, u.user_timezone, p.theme->>'theme'
  FROM profiles p JOIN users u ON u.id = p.user_id WHERE u.is_bot = false ORDER BY u.email;
SELECT 'bot', u.id, u.username, u.first_name, u.last_name, u.display_name FROM users u WHERE u.is_bot = true;
SELECT 'espaco', w.id, w.slug, w.timezone FROM workspaces w WHERE w.deleted_at IS NULL;
SELECT 'projeto', p.id, p.identifier, p.name, p.timezone, left(p.description, 60), left(p.cover_image, 60)
  FROM projects p JOIN workspaces w ON w.id = p.workspace_id
 WHERE w.slug = 'marketing-suite' AND p.deleted_at IS NULL;
SELECT 'estado', s.id, s."group", s.name, s.slug FROM states s JOIN projects p ON p.id = s.project_id
 WHERE p.identifier = 'MS' AND s.deleted_at IS NULL ORDER BY s.sequence;
SELECT 'ciclo', c.id, c.name, c.created_by_id FROM cycles c JOIN projects p ON p.id = c.project_id
 WHERE p.identifier = 'MS' AND c.deleted_at IS NULL;
SELECT 'modulo', m.id, m.name, m.created_by_id FROM modules m JOIN projects p ON p.id = m.project_id
 WHERE p.identifier = 'MS' AND m.deleted_at IS NULL;
SELECT 'etiqueta', l.id, l.name, l.created_by_id FROM labels l JOIN projects p ON p.id = l.project_id
 WHERE p.identifier = 'MS' AND l.deleted_at IS NULL;
SELECT 'visao', v.id, v.name, v.created_by_id FROM issue_views v JOIN projects p ON p.id = v.project_id
 WHERE p.identifier = 'MS' AND v.deleted_at IS NULL;
SELECT 'pagina', g.id, g.name, g.access, g.owned_by_id FROM pages g
  JOIN project_pages pp ON pp.page_id = g.id JOIN projects p ON p.id = pp.project_id
 WHERE p.identifier = 'MS' AND g.deleted_at IS NULL AND pp.deleted_at IS NULL;
