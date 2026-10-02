-- Milestone 3 schema: spoof_attempts (anti-spoofing audit log).
-- Every rejected liveness check is recorded here. Repeated failures
-- within the lockout window temporarily lock face verification
-- for the employee (enforced in the service layer).

CREATE TABLE IF NOT EXISTS spoof_attempts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    employee_id BIGINT UNSIGNED NOT NULL,
    liveness_score FLOAT NOT NULL,
    reason VARCHAR(255) NOT NULL,
    device_id VARCHAR(128) NULL DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_spoof_attempts_employee FOREIGN KEY (employee_id)
        REFERENCES employees (id) ON DELETE CASCADE,
    KEY idx_spoof_attempts_employee_time (employee_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
