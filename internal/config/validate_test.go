package config

import "testing"

func TestValidatePassword(t *testing.T) {
	valid := []string{"Password1", "abcDEF123"}
	for _, password := range valid {
		if err := ValidatePassword(password); err != nil {
			t.Fatalf("expected %q to be valid: %v", password, err)
		}
	}

	invalid := []string{"short1", "password!", "中文Password1", "has space1"}
	for _, password := range invalid {
		if err := ValidatePassword(password); err == nil {
			t.Fatalf("expected %q to be invalid", password)
		}
	}
}
