package vault

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dek, err := NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCipher(dek)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello markdown body")
	blob, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Decrypt(blob)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("got %q", out)
	}
}

func TestWrapDEKWrongPassword(t *testing.T) {
	dek, _ := NewDEK()
	kdf, _ := NewKDFParamsSalt()
	kek := DeriveKEK("correct", kdf)
	wrapped, err := WrapDEK(kek, dek)
	if err != nil {
		t.Fatal(err)
	}
	wrong := DeriveKEK("wrong", kdf)
	if _, err := UnwrapDEK(wrong, wrapped); err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
}
