-- Store-scoped global A4 document template and design settings.
ALTER TABLE store_receipt_settings
    ADD COLUMN IF NOT EXISTS document_template_key VARCHAR(32) NOT NULL DEFAULT 'standard',
    ADD COLUMN IF NOT EXISTS document_design JSONB NOT NULL DEFAULT '{"primary_color":"#1a1a1a","font_size":11,"alignment":"left","margin_mm":12,"show_logo":true,"show_store_info":true,"spacing":"standard"}'::jsonb;
