package creditsale

import "pos-backend/internal/idgen"

func newCreditSaleID() string { return idgen.Generate(idgen.PrefixCreditSale) }

func newCreditPaymentID() string { return idgen.Generate(idgen.PrefixCreditPayment) }
