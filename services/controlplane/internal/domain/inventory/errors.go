package inventory

import "errors"

var ErrNotFound = errors.New("not found")
var ErrInvalid = errors.New("invalid input")
var ErrEnrollmentInvalid = errors.New("enrollment token invalid")
var ErrEnrollmentExhausted = errors.New("enrollmenttoken exhausted")
var ErrDeviceExists = errors.New("device already enrolled")
var ErrAlreadyMember = errors.New("device already in group")
