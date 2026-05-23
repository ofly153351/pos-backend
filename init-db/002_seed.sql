-- ============================================================
-- Seed Data — Development / Demo
-- ============================================================

-- ── USERS ────────────────────────────────────────────────────
-- Passwords (plain):
--   admin@pos.dev    → Admin1234!
--   owner@pos.dev    → Owner1234!
--   manager@pos.dev  → Manager1234!
--   cashier@pos.dev  → Cashier1234!

INSERT INTO users (id, full_name, email, password_hash, role, status, created_at, updated_at) VALUES
  ('usr-00000001', 'Platform Admin',  'admin@pos.dev',   'ae47142e040fc0f74f307d72a62bdd6a:afb518271a1cc8414ff9a48c02c6cd8cda3e6da3773a0e7b28c7836f87f3ff8c',   'platform_admin', 'active', NOW(), NOW()),
  ('usr-00000002', 'Store Owner',     'owner@pos.dev',   '9e62be876d0fd85e1f267ad06a619745:ecfac8289b777e962d5598a54363b7e5f3bcb22150f516599894d981fb22d876',   'owner',          'active', NOW(), NOW()),
  ('usr-00000003', 'Store Manager',   'manager@pos.dev', '10502513877adafafd81d4e6cfd1902b:c7f7a5d703c58c4298a222efe37e06dea56be8d5ee99ffdbe639cd15e977bf9e', 'manager',        'active', NOW(), NOW()),
  ('usr-00000004', 'Cashier One',     'cashier@pos.dev', '19557e05a43d009ff7ebed80e9d2dc9f:b1a590b065f3705446529daa507a5343cbf2ab2312f8fd1a20856823508f75d2', 'cashier',        'active', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

-- ── STORE ────────────────────────────────────────────────────
INSERT INTO stores (id, owner_user_id, name, phone, address, currency_code, created_at, updated_at) VALUES
  ('str-00000001', 'usr-00000002', 'POS Demo Store', '02-000-0000', '123 ถนนสุขุมวิท กรุงเทพฯ', 'THB', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ── STORE MEMBERS ─────────────────────────────────────────────
INSERT INTO store_members (id, store_id, user_id, role, created_at) VALUES
  ('smb-00000001', 'str-00000001', 'usr-00000002', 'owner',   NOW()),
  ('smb-00000002', 'str-00000001', 'usr-00000003', 'manager', NOW()),
  ('smb-00000003', 'str-00000001', 'usr-00000004', 'cashier', NOW())
ON CONFLICT (store_id, user_id) DO NOTHING;

-- ── SUBSCRIPTION ─────────────────────────────────────────────
INSERT INTO store_subscriptions (id, store_id, plan_id, status, current_period_start, current_period_end, created_at) VALUES
  ('sub-00000001', 'str-00000001', 'plan_growth_monthly', 'active', NOW(), NOW() + INTERVAL '30 days', NOW())
ON CONFLICT DO NOTHING;

-- ── PRODUCT UNITS ─────────────────────────────────────────────
INSERT INTO product_units (id, store_id, name, is_active, created_at, updated_at) VALUES
  ('punit-00000001', 'str-00000001', 'ชิ้น',    TRUE, NOW(), NOW()),
  ('punit-00000002', 'str-00000001', 'กล่อง',   TRUE, NOW(), NOW()),
  ('punit-00000003', 'str-00000001', 'แพ็ค',    TRUE, NOW(), NOW()),
  ('punit-00000004', 'str-00000001', 'โหล',     TRUE, NOW(), NOW()),
  ('punit-00000005', 'str-00000001', 'ขวด',     TRUE, NOW(), NOW()),
  ('punit-00000006', 'str-00000001', 'ถุง',     TRUE, NOW(), NOW())
ON CONFLICT (store_id, name) DO NOTHING;

-- ── PRODUCT TYPES ─────────────────────────────────────────────
INSERT INTO product_types (id, store_id, name, is_active, created_at, updated_at) VALUES
  ('ptype-00000001', 'str-00000001', 'เครื่องดื่ม',  TRUE, NOW(), NOW()),
  ('ptype-00000002', 'str-00000001', 'ขนม',           TRUE, NOW(), NOW()),
  ('ptype-00000003', 'str-00000001', 'ของใช้',        TRUE, NOW(), NOW()),
  ('ptype-00000004', 'str-00000001', 'อาหาร',         TRUE, NOW(), NOW())
ON CONFLICT (store_id, name) DO NOTHING;

-- ── PRODUCT BRANDS ────────────────────────────────────────────
INSERT INTO product_brands (id, store_id, name, is_active, created_at, updated_at) VALUES
  ('brand-0000001', 'str-00000001', 'เนสท์เล่',    TRUE, NOW(), NOW()),
  ('brand-0000002', 'str-00000001', 'โค้ก',         TRUE, NOW(), NOW()),
  ('brand-0000003', 'str-00000001', 'เลย์',          TRUE, NOW(), NOW()),
  ('brand-0000004', 'str-00000001', 'โนแบรนด์',     TRUE, NOW(), NOW())
ON CONFLICT (store_id, name) DO NOTHING;

-- ── PRODUCTS ──────────────────────────────────────────────────
INSERT INTO products (id, store_id, product_type_id, product_unit_id, brand_id, name, sku, barcode, base_price, cost_price, min_stock, is_active, created_at, updated_at) VALUES
  ('pd-00000001', 'str-00000001', 'ptype-00000001', 'punit-00000001', 'brand-0000002', 'โค้ก 325ml',           'COKE-325',    '8850999226271', 20.00,  12.00, 10, TRUE, NOW(), NOW()),
  ('pd-00000002', 'str-00000001', 'ptype-00000001', 'punit-00000001', 'brand-0000002', 'โค้ก 1.25L',           'COKE-125L',   '8850999226318', 35.00,  22.00, 10, TRUE, NOW(), NOW()),
  ('pd-00000003', 'str-00000001', 'ptype-00000001', 'punit-00000005', 'brand-0000001', 'เนสกาแฟ กระป๋อง 180ml','NESCAFE-180', '8850290022438', 18.00,  11.00, 20, TRUE, NOW(), NOW()),
  ('pd-00000004', 'str-00000001', 'ptype-00000001', 'punit-00000006', 'brand-0000004', 'น้ำดื่ม 600ml',        'WATER-600',   '8851234567890', 10.00,   5.00, 50, TRUE, NOW(), NOW()),
  ('pd-00000005', 'str-00000001', 'ptype-00000002', 'punit-00000001', 'brand-0000003', 'เลย์ ออริจินัล 75g',   'LAY-ORIG-75', '8850718111219', 25.00,  16.00, 20, TRUE, NOW(), NOW()),
  ('pd-00000006', 'str-00000001', 'ptype-00000002', 'punit-00000001', 'brand-0000003', 'เลย์ บาร์บีคิว 75g',   'LAY-BBQ-75',  '8850718111226', 25.00,  16.00, 20, TRUE, NOW(), NOW()),
  ('pd-00000007', 'str-00000001', 'ptype-00000002', 'punit-00000001', 'brand-0000004', 'คุกกี้ช็อกโกแลต',      'COOKIE-CHOC', '8851111111111', 35.00,  20.00, 15, TRUE, NOW(), NOW()),
  ('pd-00000008', 'str-00000001', 'ptype-00000003', 'punit-00000001', 'brand-0000004', 'สบู่ก้อน',             'SOAP-BAR-01', '8852222222222', 29.00,  15.00, 10, TRUE, NOW(), NOW()),
  ('pd-00000009', 'str-00000001', 'ptype-00000003', 'punit-00000006', 'brand-0000004', 'แชมพู 200ml',          'SHAMPOO-200', '8853333333333', 89.00,  55.00,  5, TRUE, NOW(), NOW()),
  ('pd-00000010', 'str-00000001', 'ptype-00000004', 'punit-00000001', 'brand-0000004', 'มาม่า ต้มยำกุ้ง',      'MAMA-TYK',   '8851503101237', 8.00,   4.50, 30, TRUE, NOW(), NOW()),
  ('pd-00000011', 'str-00000001', 'ptype-00000004', 'punit-00000001', 'brand-0000004', 'มาม่า หมูสับ',         'MAMA-PK',    '8851503101220', 8.00,   4.50, 30, TRUE, NOW(), NOW()),
  ('pd-00000012', 'str-00000001', 'ptype-00000004', 'punit-00000002', 'brand-0000004', 'ข้าวสาร 5kg',          'RICE-5KG',   '8854444444444', 175.00,130.00,  5, TRUE, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ── WAREHOUSE ─────────────────────────────────────────────────
INSERT INTO warehouses (id, store_id, name, code, is_active, created_at, updated_at) VALUES
  ('wh-00000001', 'str-00000001', 'คลังหลัก',   'WH-MAIN', TRUE, NOW(), NOW()),
  ('wh-00000002', 'str-00000001', 'หน้าร้าน',   'WH-SHOP', TRUE, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ── LOCATIONS ─────────────────────────────────────────────────
INSERT INTO locations (id, store_id, warehouse_id, name, code, is_sale_point, is_active, created_at, updated_at) VALUES
  ('loc-00000001', 'str-00000001', 'wh-00000001', 'ชั้นวาง A',    'LOC-A',  FALSE, TRUE, NOW(), NOW()),
  ('loc-00000002', 'str-00000001', 'wh-00000001', 'ชั้นวาง B',    'LOC-B',  FALSE, TRUE, NOW(), NOW()),
  ('loc-00000003', 'str-00000001', 'wh-00000002', 'เคาน์เตอร์',  'COUNTER',TRUE,  TRUE, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ── STOCKS ────────────────────────────────────────────────────
INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at) VALUES
  ('stk-00000001', 'str-00000001', 'pd-00000001', 'loc-00000003', 50,  NOW(), NOW()),
  ('stk-00000002', 'str-00000001', 'pd-00000002', 'loc-00000003', 30,  NOW(), NOW()),
  ('stk-00000003', 'str-00000001', 'pd-00000003', 'loc-00000003', 40,  NOW(), NOW()),
  ('stk-00000004', 'str-00000001', 'pd-00000004', 'loc-00000003', 100, NOW(), NOW()),
  ('stk-00000005', 'str-00000001', 'pd-00000005', 'loc-00000003', 60,  NOW(), NOW()),
  ('stk-00000006', 'str-00000001', 'pd-00000006', 'loc-00000003', 60,  NOW(), NOW()),
  ('stk-00000007', 'str-00000001', 'pd-00000007', 'loc-00000003', 25,  NOW(), NOW()),
  ('stk-00000008', 'str-00000001', 'pd-00000008', 'loc-00000003', 20,  NOW(), NOW()),
  ('stk-00000009', 'str-00000001', 'pd-00000009', 'loc-00000003', 15,  NOW(), NOW()),
  ('stk-00000010', 'str-00000001', 'pd-00000010', 'loc-00000003', 80,  NOW(), NOW()),
  ('stk-00000011', 'str-00000001', 'pd-00000011', 'loc-00000003', 80,  NOW(), NOW()),
  ('stk-00000012', 'str-00000001', 'pd-00000012', 'loc-00000001', 10,  NOW(), NOW()),
  -- คลังสำรองใน WH-MAIN
  ('stk-00000013', 'str-00000001', 'pd-00000001', 'loc-00000001', 200, NOW(), NOW()),
  ('stk-00000014', 'str-00000001', 'pd-00000004', 'loc-00000001', 300, NOW(), NOW()),
  ('stk-00000015', 'str-00000001', 'pd-00000010', 'loc-00000001', 150, NOW(), NOW()),
  ('stk-00000016', 'str-00000001', 'pd-00000011', 'loc-00000001', 150, NOW(), NOW())
ON CONFLICT (product_id, location_id) DO NOTHING;
