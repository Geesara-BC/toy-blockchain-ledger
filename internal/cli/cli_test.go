package cli

import "testing"

func TestNormalizeWalletName(t *testing.T) {
	got := normalizeWalletName("  Janindu  ")
	want := "janindu"
	if got != want {
		t.Fatalf("normalizeWalletName() = %q, want %q", got, want)
	}
}

func TestHasWalletNameTreatsDuplicateNamesAsCaseInsensitive(t *testing.T) {
	addressBook := map[string]string{"janindu": "pubkey"}

	if !hasWalletName(addressBook, "JANINDU") {
		t.Fatalf("expected duplicate wallet name to be detected")
	}

	if !hasWalletName(addressBook, " janindu ") {
		t.Fatalf("expected trimmed duplicate wallet name to be detected")
	}
}
