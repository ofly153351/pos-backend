package expense

import "pos-backend/internal/idgen"

func newExpenseID() string { return idgen.Generate(idgen.PrefixExpense) }

func newCategoryID() string { return idgen.Generate(idgen.PrefixExpenseCategory) }
