package crypto

import "testing"

func TestPasswordHashAndDerivedKey(t *testing.T) {
	hash, err := HashPassword("Password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !VerifyPassword("Password123", hash) {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword("WrongPassword", hash) {
		t.Fatal("wrong password verified")
	}
	if RandomPasswordKey("Password123", "shared-salt") != RandomPasswordKey("Password123", "shared-salt") {
		t.Fatal("derived password key should be stable for same password and salt")
	}
}
