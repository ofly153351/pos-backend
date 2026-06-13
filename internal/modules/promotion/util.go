package promotion

import "pos-backend/internal/idgen"

func newPromotionID() string { return idgen.Generate(idgen.PrefixPromotion) }
