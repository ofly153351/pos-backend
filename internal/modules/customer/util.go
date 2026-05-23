package customer

import "pos-backend/internal/idgen"

func newID() string { return idgen.Generate(idgen.PrefixCustomer) }
