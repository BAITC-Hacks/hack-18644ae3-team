-- Keep human-readable employee IDs while allocating them safely across requests.
CREATE SEQUENCE IF NOT EXISTS employee_id_seq START WITH 1;

SELECT setval(
    'employee_id_seq',
    GREATEST(COALESCE((SELECT MAX(SUBSTRING(id FROM 2)::BIGINT) FROM employees WHERE id ~ '^E[0-9]+$'), 0), 1),
    EXISTS(SELECT 1 FROM employees WHERE id ~ '^E[0-9]+$')
);
