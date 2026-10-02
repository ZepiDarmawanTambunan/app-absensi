-- Milestone 2 schema: face_enrollments (on-device embeddings).
-- The server stores embedding templates only; raw face photos are never stored.

CREATE TABLE IF NOT EXISTS face_enrollments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    employee_id BIGINT UNSIGNED NOT NULL,
    embedding LONGBLOB NOT NULL,
    dimension INT NOT NULL,
    quality_score FLOAT NOT NULL,
    enrolled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    CONSTRAINT fk_face_enrollments_employee FOREIGN KEY (employee_id)
        REFERENCES employees (id) ON DELETE CASCADE,
    KEY idx_face_enrollments_employee (employee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
