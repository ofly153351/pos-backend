package parkedbill

import "pos-backend/internal/idgen"

func newID() string     { return idgen.Generate(idgen.PrefixParkedBill) }
func newItemID() string { return idgen.Generate(idgen.PrefixParkedBillItem) }
