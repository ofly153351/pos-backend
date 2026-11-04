-- B03: snapshot the buyer's branch on documents so tax invoices issued to a
-- company show its tax ID together with the branch (Revenue Code §86/4).
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS customer_branch TEXT;
