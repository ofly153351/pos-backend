-- Normalize legacy expense payment_method keys to the six canonical keys.
-- Idempotent: re-running only affects rows still on the old vocabulary.
UPDATE expenses SET payment_method = 'bank_transfer' WHERE payment_method = 'transfer';
UPDATE expenses SET payment_method = 'promptpay'     WHERE payment_method = 'qr';
UPDATE expenses SET payment_method = 'credit_card'   WHERE payment_method = 'credit';
UPDATE expenses SET payment_method = 'debit_card'    WHERE payment_method = 'card';
