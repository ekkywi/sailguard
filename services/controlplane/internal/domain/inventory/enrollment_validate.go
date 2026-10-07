package inventory

import "time"

func (t *EnrollmentToken) ValidateForEnroll(now time.Time) error {
	if t.RevokedAt != nil {
		return ErrEnrollmentInvalid
	}
	if t.ExpiresAt != nil && !t.ExpiresAt.After(now) {
		return ErrEnrollmentInvalid
	}
	if t.UseCount >= t.MaxUses {
		return ErrEnrollmentExhausted
	}
	return nil
}