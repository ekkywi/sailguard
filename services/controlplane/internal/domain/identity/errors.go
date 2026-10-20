package identity

import "errors"

var (
	ErrNotFound       = errors.New("user not found")
	ErrConflict       = errors.New("user conflict")
	ErrInvalidRole    = errors.New("invalid role")
	ErrLastSuperAdmin = errors.New("last super_admin")
)
