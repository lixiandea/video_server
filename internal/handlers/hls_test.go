package handlers

import "testing"

func TestValidHLSFileName(t *testing.T) {
	for _, name := range []string{"index.m3u8", "480p.m3u8", "720p_000.ts"} {
		if !validHLSFileName(name) {
			t.Errorf("期望 %q 合法", name)
		}
	}

	for _, name := range []string{"", "../index.m3u8", "a/b.ts", "a\\b.ts", "video.mp4", "..ts"} {
		if validHLSFileName(name) {
			t.Errorf("期望 %q 非法", name)
		}
	}
}

func TestWithPlaylistToken(t *testing.T) {
	content := []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=900000\n480p.m3u8\n\n720p.m3u8?x=1\n")
	got := string(withPlaylistToken(content, "abc.def"))
	want := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=900000\n480p.m3u8?token=abc.def\n\n720p.m3u8?x=1\n"
	if got != want {
		t.Fatalf("playlist 改写错误:\n got: %q\nwant: %q", got, want)
	}
}
