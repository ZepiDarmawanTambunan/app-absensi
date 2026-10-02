-- Roll back Milestone 1 schema (reverse dependency order).

DROP TABLE IF EXISTS attendance_logs;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS users;
