package main

import "testing"

func TestIsChangesMatch(t *testing.T) {
	tests := []struct {
		rule    *Rule
		changes []string
		match   bool
	}{
		{
			rule: &Rule{
				Path: "abcd",
			},
			changes: []string{"foobar"},
			match:   false,
		},
		{
			rule: &Rule{
				Path: "foobar",
			},
			changes: []string{"foobar"},
			match:   true,
		},
		{
			rule: &Rule{
				Path: "foobar/**",
			},
			changes: []string{"foobar/a", "foobar/b"},
			match:   true,
		},
	}
	for _, test := range tests {
		_, match := test.rule.IsChangesMatch(test.changes)
		if match != test.match {
			t.Errorf("Must be %v, but got %v", test.match, match)
		}
	}
}
