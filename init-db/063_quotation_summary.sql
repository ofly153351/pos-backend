ALTER TABLE documents
  ADD COLUMN IF NOT EXISTS quotation_summary TEXT;
