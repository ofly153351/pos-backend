-- Compact display format: QUO/2569/09/0001 -> QUO256909-0001.
-- Keep the sequence semantics unchanged; both fields use the compact display format.
UPDATE documents
SET document_no_full = regexp_replace(
    document_no_full,
    '^([^/]+)/([0-9]{4})/([0-9]{2})/([0-9]{4})$',
    '\1\2\3-\4'
)
WHERE document_no_full ~ '^([^/]+)/([0-9]{4})/([0-9]{2})/([0-9]{4})$';

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