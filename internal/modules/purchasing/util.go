package purchasing

import "pos-backend/internal/idgen"

func newID() string           { return idgen.Generate(idgen.PrefixPurchaseOrder) }
func newSupplierID() string   { return idgen.Generate(idgen.PrefixSupplier) }
func newPOItemID() string     { return idgen.Generate(idgen.PrefixPurchaseOrderItem) }
func newLocationID() string   { return idgen.Generate(idgen.PrefixLocation) }
func newStockID() string      { return idgen.Generate(idgen.PrefixStock) }
func newProductID() string    { return idgen.Generate(idgen.PrefixProduct) }
func newProductUnitID() string { return idgen.Generate(idgen.PrefixProductUnit) }
func newSupplierProductID() string { return idgen.Generate(idgen.PrefixSupplierProduct) }
