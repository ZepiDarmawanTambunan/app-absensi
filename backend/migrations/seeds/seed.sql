-- Seed data for local development. Apply AFTER migrations:
--   mysql absensi_db < migrations/seeds/seed.sql
--
-- Demo operator: admin@example.com / admin123
-- Change or remove these credentials before any shared/staging use.

INSERT INTO users (name, email, password_hash, role)
VALUES ('Admin', 'admin@example.com', '$2a$10$Cx2J6xC8/Zs4ngLev3tE0ugioo9SUDstvY/iuj51rM6uftfVQm7zm', 'admin')
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO employees (employee_no, name, department, is_active)
VALUES
    ('EMP001', 'Budi Santoso', 'IT', TRUE),
    ('EMP002', 'Siti Rahma', 'HR', TRUE),
    ('EMP003', 'Andi Pratama', 'Operasional', TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name), department = VALUES(department);
