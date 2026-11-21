package qrcode

import "strings"
import "testing"

func TestModules(t *testing.T) {
	m := Modules("http://192.168.178.58:8765/")
	if m == nil || len(m) != 25 {
		t.Fatalf("version 2, got %d", len(m))
	}
	if Modules("") != nil || Modules(strings.Repeat("x", 54)) != nil {
		t.Fatal("bounds")
	}
	if m3 := Modules(strings.Repeat("x", 40)); m3 == nil || len(m3) != 29 {
		t.Fatal("version 3")
	}
}
