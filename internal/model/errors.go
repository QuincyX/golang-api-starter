package model

import "errors"

// Domain-level errors shared by store, service and API mapping. Keeping them on
// the model avoids API packages importing the store just to compare errors.
var (
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrLastSuperAdmin = errors.New("cannot remove or disable the last super admin")
)
