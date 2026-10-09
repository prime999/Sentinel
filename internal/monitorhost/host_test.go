package monitorhost

import "testing"

func TestNormalizeHost(t *testing.T) {
	cases := map[string]string{
		"https://www.example.com/path": "www.example.com",
		"http://EXAMPLE.com":         "example.com",
		"example.com":                "example.com",
		"  https://shop.test:443/  ": "shop.test",
	}
	for in, want := range cases {
		if got := NormalizeHost(in); got != want {
			t.Fatalf("%q => %q want %q", in, got, want)
		}
	}
}
