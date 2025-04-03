package defaulttgo

import "testing"

func ExpectDisabled(t *testing.T) {
	t.Helper()
	if Enabled {
		t.Skip("test skip in -tags defaulttgo mode")
	}
}
