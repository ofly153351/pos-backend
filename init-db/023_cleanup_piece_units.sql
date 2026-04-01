INSERT INTO product_units (id, store_id, name, description, is_active, created_at, updated_at)
SELECT DISTINCT
    'pu_' || SUBSTR(MD5(pu.store_id || ':unit'), 1, 24),
    pu.store_id,
    'unit',
    NULL,
    TRUE,
    NOW(),
    NOW()
FROM product_units pu
WHERE LOWER(TRIM(pu.name)) LIKE 'piece%'
ON CONFLICT (id) DO NOTHING;

UPDATE products p
SET product_unit_id = 'pu_' || SUBSTR(MD5(p.store_id || ':unit'), 1, 24)
FROM product_units pu
WHERE pu.id = p.product_unit_id
  AND LOWER(TRIM(pu.name)) LIKE 'piece%';

DELETE FROM product_units pu
WHERE LOWER(TRIM(pu.name)) LIKE 'piece%';
