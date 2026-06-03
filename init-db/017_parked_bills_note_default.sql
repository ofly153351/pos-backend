ALTER TABLE parked_bills ALTER COLUMN note SET DEFAULT '';
UPDATE parked_bills SET note = '' WHERE note IS NULL;
ALTER TABLE parked_bills ALTER COLUMN note SET NOT NULL;
