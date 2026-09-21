-- Compact display format: QUO/2569/09/0001 -> QUO256909-0001.
-- Keep document_no unchanged; it remains the internal sequence identifier.
UPDATE documents
SET document_no_full = regexp_replace(
    document_no_full,
    '^([^/]+)/([0-9]{4})/([0-9]{2})/([0-9]{4})$',
    '\1\2\3-\4'
)
WHERE document_no_full ~ '^([^/]+)/([0-9]{4})/([0-9]{2})/([0-9]{4})$';