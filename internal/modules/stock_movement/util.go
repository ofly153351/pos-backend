package stock_movement

import "pos-backend/internal/idgen"

func newID() string      { return idgen.Generate(idgen.PrefixStockMovement) }
func newStockID() string { return idgen.Generate(idgen.PrefixStock) }
