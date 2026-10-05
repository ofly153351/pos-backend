-- Immutable snapshots of document state before each edit/restore.
CREATE TABLE IF NOT EXISTS document_revisions (
    id            VARCHAR(30) PRIMARY KEY,
    document_id   VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    revision_no   INTEGER NOT NULL,
    action        VARCHAR(20) NOT NULL DEFAULT 'EDIT',
    snapshot      JSONB NOT NULL,
    changed_by    VARCHAR(30) NOT NULL,
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (document_id, revision_no)
);

CREATE INDEX IF NOT EXISTS idx_document_revisions_document
    ON document_revisions(document_id, revision_no DESC);
