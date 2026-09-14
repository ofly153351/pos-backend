package htmlpdf

import "testing"

func TestResolveExecPath_EnvOverride(t *testing.T) {
	t.Setenv("CHROME_PATH", "/tmp/test-browser")
	if got := resolveExecPath(); got != "/tmp/test-browser" {
		t.Fatalf("path = %q, want env override", got)
	}
}

func TestResolveExecPath_FindsMacBrowser(t *testing.T) {
	t.Setenv("CHROME_PATH", "")
	got := resolveExecPath()
	if got == "" {
		t.Skip("no supported browser installed on this host")
	}
}
