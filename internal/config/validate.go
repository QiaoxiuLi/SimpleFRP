package config

import "regexp"

var passwordRe = regexp.MustCompile(`^[A-Za-z0-9]{8,}$`)

func ValidatePassword(password string) error {
	if !passwordRe.MatchString(password) {
		return ErrInvalidPassword
	}
	return nil
}

var ErrInvalidPassword = simpleError("Invalid password. Password must be at least 8 characters long and contain only letters and numbers. Please try again.")

type simpleError string

func (e simpleError) Error() string { return string(e) }
