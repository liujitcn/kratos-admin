package oauth

import "testing"

func TestMatchClientIP(t *testing.T) {
	tests := []struct {
		name      string
		clientIP  string
		whitelist string
		want      bool
	}{
		{name: "exact ipv4", clientIP: "192.0.2.10", whitelist: "192.0.2.10", want: true},
		{name: "cidr", clientIP: "192.0.2.10", whitelist: "192.0.2.0/24", want: true},
		{name: "outside cidr", clientIP: "192.0.3.10", whitelist: "192.0.2.0/24", want: false},
		{name: "multiple entries", clientIP: "2001:db8::2", whitelist: "192.0.2.0/24, 2001:db8::/64", want: true},
		{name: "invalid entry", clientIP: "192.0.2.10", whitelist: "not-an-ip", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MatchClientIP(test.clientIP, test.whitelist); got != test.want {
				t.Fatalf("MatchClientIP(%q, %q) = %v, want %v", test.clientIP, test.whitelist, got, test.want)
			}
		})
	}
}
