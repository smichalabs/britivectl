package britive

import "testing"

func TestSanitizeTenant(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"acme", "acme"},
		{"ACME", "acme"},
		{"  acme  ", "acme"},
		{"acme.britive-app.com", "acme"},
		{"https://acme.britive-app.com", "acme"},
		{"http://acme.britive-app.com", "acme"},
		{"https://acme.britive-app.com/", "acme"},
		{"https://ACME.britive-app.com/", "acme"},
		{"", ""},
		{"acme-prod", "acme-prod"},
	}
	for _, c := range cases {
		if got := SanitizeTenant(c.in); got != c.want {
			t.Errorf("SanitizeTenant(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
