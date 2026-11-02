package listquery

import "testing"

func TestOrderBy(t *testing.T) {
	allowed := map[string]string{
		"hostname": "hostname",
		"created":  "created_at",
	}
	const def = "hostname ASC"

	tests := []struct {
		sort, order, want string
	}{
		{"", "", def},
		{"unknown", "desc", def},
		{"hostname", "", "hostname ASC"},
		{"hostname", "asc", "hostname ASC"},
		{"hostname", "DESC", "hostname DESC"},
		{"Created", "desc", "created_at DESC"},
	}
	for _, tc := range tests {
		got := OrderBy(tc.sort, tc.order, allowed, def)
		if got != tc.want {
			t.Errorf("OrderBy(%q,%q)=%q want %q", tc.sort, tc.order, got, tc.want)
		}
	}
}
