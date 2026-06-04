package daemon

import "testing"

func TestSupportsMediaSend(t *testing.T) {
	cases := []struct {
		name     string
		features []string
		want     bool
	}{
		{"advertised", []string{FeatureMediaSend}, true},
		{"advertised among others", []string{"other", FeatureMediaSend}, true},
		{"old daemon omits it", []string{"other"}, false},
		{"old daemon no features", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{hello: HelloPayload{Features: tc.features}}
			if got := c.SupportsMediaSend(); got != tc.want {
				t.Fatalf("SupportsMediaSend() = %v, want %v", got, tc.want)
			}
		})
	}
}
