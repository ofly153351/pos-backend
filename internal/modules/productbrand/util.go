package productbrand

import "pos-backend/internal/idgen"

func newID() string { return idgen.Generate(idgen.PrefixProductBrand) }
