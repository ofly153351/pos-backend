-- Align the short document number with the compact display format.
-- QUO-2609-0009 -> QUO256909-0009
-- The migration is idempotent because it only matches the legacy pattern.
WITH parsed AS (
    SELECT id, regexp_matches(
        document_no,
        '^([A-Z]+)-([0-9]{2})([0-9]{2})-([0-9]{4})$'
    ) AS parts
    FROM documents
)
UPDATE documents d
SET document_no = p.parts[1]
    || (2543 + p.parts[2]::INTEGER)::TEXT
    || p.parts[3]
    || '-'
    || p.parts[4]
FROM parsed p
WHERE d.id = p.id;
