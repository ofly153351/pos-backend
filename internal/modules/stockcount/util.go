package stockcount

import "pos-backend/internal/idgen"

func newSessionID() string { return idgen.Generate(idgen.PrefixStockCountSession) }

func newItemID() string { return idgen.Generate(idgen.PrefixStockCountItem) }
