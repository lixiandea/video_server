package auth

import "testing"

func TestRefreshTokenGenerationUsesOneWayHash(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" || raw == hash {
		t.Fatalf("unexpected refresh token values: raw=%q hash=%q", raw, hash)
	}
	if got := HashRefreshToken(raw); got != hash {
		t.Fatalf("hash mismatch: got %s want %s", got, hash)
	}
	if EqualTokenHash(hash, HashRefreshToken("other")) {
		t.Fatal("different refresh tokens must not compare equal")
	}
}

func TestInitTokenTTLRejectsInvalidPolicy(t *testing.T) {
	if err := InitTokenTTL(0, RefreshTokenTTL); err == nil {
		t.Fatal("expected invalid access TTL to fail")
	}
	if err := InitTokenTTL(RefreshTokenTTL, RefreshTokenTTL); err == nil {
		t.Fatal("expected refresh TTL equal to access TTL to fail")
	}
}
