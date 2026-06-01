ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_document_id VARCHAR(30) REFERENCES documents(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_documents_source_document ON documents(source_document_id) WHERE source_document_id IS NOT NULL;
