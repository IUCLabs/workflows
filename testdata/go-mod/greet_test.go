package gomod

import "testing"

func TestGreet(t *testing.T) {
	for in, want := range map[string]string{"": "hello, world", "bob": "hello, bob"} {
		if got := Greet(in); got != want {
			t.Errorf("Greet(%q) = %q, want %q", in, got, want)
		}
	}
}
