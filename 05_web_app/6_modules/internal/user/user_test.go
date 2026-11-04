package user

import (
	"strconv"
	"testing"
)

func TestIsOdd(t *testing.T) {
	cases := []struct {
		id   uint32
		want bool
	}{
		{id: 1, want: true},
		{id: 2, want: false},
	}

	for _, tc := range cases {
		t.Run(strconv.FormatUint(uint64(tc.id), 10), func(t *testing.T) {
			u := NewUser(tc.id, "test")
			if got := u.IsOdd(); got != tc.want {
				t.Errorf("IsOdd() = %v, want %v", got, tc.want)
			}
		})
	}
}
