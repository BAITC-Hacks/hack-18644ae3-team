-- A self-reported employee ID is a request, not an approved identity link.
ALTER TABLE users ADD COLUMN requested_employee_id TEXT NOT NULL DEFAULT '';

UPDATE users
SET requested_employee_id = COALESCE(employee_id, ''), employee_id = NULL
WHERE status IN ('PENDING', 'REJECTED');
