package main

import "testing"

func TestGetNormalizedExtension(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{{"", ""}, {"a", ""}, {"a.b", "b"}, {"a.B", "b"}}

	for _, c := range cases {
		got := GetNormalizedExtension(c.path)
		if got != c.want {
			t.Errorf("GetNormalizedExtension(%q) == %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsHidden(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{{"", false}, {"a", false}, {"a.b", false}, {".a", true}}

	for _, c := range cases {
		got := IsHidden(c.name)
		if got != c.want {
			t.Errorf("TestIsHidden(%q) == %v, want %v", c.name, got, c.want)
		}
	}
}
