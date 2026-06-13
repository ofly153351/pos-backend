-- 031_store_members_role_status.sql
-- P1.2 / P2.4 / P2.5: store-staff management needs two things the schema lacks.
--
-- 1) A 'warehouse' store role. store_members.role originally allowed only
--    ('owner','manager','cashier'). Warehouse staff receive stock but make no
--    purchasing/management decisions, so they are a distinct operate-level role.
--    The global users.role CHECK is intentionally left unchanged — a member's
--    store role lives only in store_members; users.role stays 'cashier'.
--
-- 2) A per-store member status ('active'/'suspended'). Status belongs on the
--    membership, not on users, because one user may belong to several stores and
--    suspending them in store A must not lock them out of store B.
--
-- Idempotent: DROP/ADD the named CHECK constraint, ADD COLUMN IF NOT EXISTS.

ALTER TABLE store_members DROP CONSTRAINT IF EXISTS store_members_role_check;
ALTER TABLE store_members
    ADD CONSTRAINT store_members_role_check
    CHECK (role IN ('owner','manager','cashier','warehouse'));

ALTER TABLE store_members
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','suspended'));
