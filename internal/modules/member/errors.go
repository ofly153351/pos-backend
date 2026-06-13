package member

import "errors"

var (
	ErrStoreIDRequired = errors.New("storeID is required")
	ErrForbidden       = errors.New("you do not have permission to manage staff in this store")
	ErrMemberNotFound  = errors.New("member not found")
	ErrAlreadyMember   = errors.New("this user is already a member of the store")
	ErrInvalidRole     = errors.New("role must be one of owner, manager, cashier, warehouse")
	ErrInvalidStatus   = errors.New("status must be active or suspended")
	ErrOwnerOnly       = errors.New("only an owner can grant or modify the owner role")
	ErrLastOwner       = errors.New("cannot remove, demote, or suspend the last active owner")
	ErrCannotSelfRemove = errors.New("you cannot remove your own membership")
	ErrCannotSelfSuspend = errors.New("you cannot suspend your own membership")
	ErrNameRequired    = errors.New("name is required for a new user")
	ErrEmailRequired   = errors.New("a valid email is required")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters for a new user")
	ErrNothingToUpdate = errors.New("nothing to update")
)
