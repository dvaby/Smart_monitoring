package security

import "testing"

func TestPasswordHasherHashAndVerify(t *testing.T) {
	hasher := NewPasswordHasher()
	encoded, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "correct horse battery staple" {
		t.Fatal("password was stored in plain text")
	}
	if !hasher.Verify(encoded, "correct horse battery staple") {
		t.Fatal("expected password to verify")
	}
	if hasher.Verify(encoded, "wrong password") {
		t.Fatal("unexpected password verification")
	}
}

func TestPasswordHasherRejectsMalformedHash(t *testing.T) {
	if NewPasswordHasher().Verify("not-a-hash", "password") {
		t.Fatal("malformed hash should be rejected")
	}
}
