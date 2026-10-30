package password

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndCompare(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)

	hash, err := hasher.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if err := hasher.Compare(hash, "password123"); err != nil {
		t.Fatalf("Compare() with the right password error = %v", err)
	}

	if err := hasher.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("Compare() with a wrong password succeeded")
	}
}
