package store

import (
	"fmt"
	"pos-backend/internal/idgen"
)

func newHexID() string              { return idgen.Generate(idgen.PrefixStore) }
func newStoreMemberID() string       { return idgen.Generate(idgen.PrefixStoreMember) }
func newStoreSubscriptionID() string { return idgen.Generate(idgen.PrefixStoreSubscription) }
func newFileToken() string           { return fmt.Sprintf("%08d", idgen.NextInt()) }
