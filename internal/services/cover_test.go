package services

import "testing"

func TestCoverVariantPath(t *testing.T) {
	got := CoverVariantPath("/app/storage/videos/abc_high_cover.jpg", 320)
	want := "/app/storage/videos/abc_high_cover_320w.jpg"
	if got != want {
		t.Fatalf("CoverVariantPath = %q, 期望 %q", got, want)
	}

	got = CoverVariantPath("storage/videos/abc_high_cover.jpeg", 720)
	want = "storage/videos/abc_high_cover_720w.jpg"
	if got != want {
		t.Fatalf("CoverVariantPath = %q, 期望 %q", got, want)
	}
}
