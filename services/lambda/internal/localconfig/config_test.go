package localconfig

import "testing"

func TestLocalOnlyConfiguration(t *testing.T) {
	for _, name := range []string{"API_ADDRESS", "WEB_ORIGIN"} {
		t.Setenv(name, "")
	}
	cfg, err := Load()
	if err != nil || cfg.Address != "127.0.0.1:8080" || cfg.Origin != "http://localhost:3000" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	for _, tc := range []struct{ name, value string }{
		{"API_ADDRESS", "0.0.0.0:8080"},
		{"API_ADDRESS", ":8080"},
		{"API_ADDRESS", "example.com:8080"},
		{"WEB_ORIGIN", "https://public.example"},
		{"WEB_ORIGIN", "http://localhost:3000/path"},
	} {
		t.Run(tc.name+tc.value, func(t *testing.T) {
			t.Setenv(tc.name, tc.value)
			if _, err := Load(); err == nil {
				t.Fatalf("accepted unsafe config %s=%s", tc.name, tc.value)
			}
		})
	}
}
