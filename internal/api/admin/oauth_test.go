package admin

import "testing"

func TestPickVerifiedGitHubEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		emails []githubEmailEntry
		want   string
	}{
		{
			name: "prefers primary verified",
			emails: []githubEmailEntry{
				{Email: "other@example.com", Verified: true},
				{Email: "primary@example.com", Primary: true, Verified: true},
			},
			want: "primary@example.com",
		},
		{
			name: "skips unverified on user endpoint fast path",
			emails: []githubEmailEntry{
				{Email: "public@example.com", Primary: true, Verified: false},
				{Email: "verified@example.com", Verified: true},
			},
			want: "verified@example.com",
		},
		{
			name: "no verified email",
			emails: []githubEmailEntry{
				{Email: "public@example.com", Primary: true, Verified: false},
			},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := pickVerifiedGitHubEmail(tc.emails); got != tc.want {
				t.Fatalf("pickVerifiedGitHubEmail() = %q, want %q", got, tc.want)
			}
		})
	}
}