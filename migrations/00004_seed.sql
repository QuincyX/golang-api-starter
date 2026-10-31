-- +goose Up
INSERT INTO permissions (code, name, grp) VALUES
    ('account:read',  '查看账号',   'account'),
    ('account:write', '管理账号',   'account'),
    ('role:read',     '查看角色',   'account'),
    ('role:write',    '管理角色',   'account'),
    ('role:assign',   '分配角色',   'account'),
    ('article:read',  '查看文章',   'article'),
    ('article:write', '编辑文章',   'article'),
    ('article:publish', '发布文章', 'article'),
    ('article:delete', '删除文章',  'article'),
    ('audit:read',    '查看审计',   'audit'),
    ('audit:export',  '导出审计',   'audit')
ON CONFLICT (code) DO NOTHING;

INSERT INTO roles (code, name, description, is_system) VALUES
    ('super_admin', '超级管理员', '拥有全部权限', true),
    ('editor',      '编辑',       '文章读写与发布', true),
    ('auditor',     '审计员',     '只读与审计', true)
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_code)
SELECT r.id, p.code
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_code)
SELECT r.id, p.code
FROM roles r
JOIN permissions p ON p.code IN ('article:read', 'article:write', 'article:publish', 'article:delete')
WHERE r.code = 'editor'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_code)
SELECT r.id, p.code
FROM roles r
JOIN permissions p ON p.code IN ('article:read', 'audit:read')
WHERE r.code = 'auditor'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE code IN ('super_admin', 'editor', 'auditor'));

DELETE FROM roles WHERE code IN ('super_admin', 'editor', 'auditor');

DELETE FROM permissions WHERE code IN (
    'account:read', 'account:write', 'role:read', 'role:write', 'role:assign',
    'article:read', 'article:write', 'article:publish', 'article:delete',
    'audit:read', 'audit:export'
);
