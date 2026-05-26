package receipt_settings

import "pos-backend/internal/idgen"

const PrefixReceiptSettings = "rset"

func newID() string { return idgen.Generate(PrefixReceiptSettings) }
