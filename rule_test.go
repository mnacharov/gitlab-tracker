package main

import (
	"reflect"
	"testing"
)

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

func TestRule_ParseAsTemplate(t *testing.T) {
	tests := []struct {
		r      Rule
		result Rule
	}{
		{
			r: Rule{
				Path:               "{{ .foo }}",
				Tag:                "{{ .foo }}",
				TagSuffix:          "{{ .foo }}",
				TagSuffixSeparator: "{{ .foo }}",
			},
			result: Rule{
				Path:               "bar",
				Tag:                "bar",
				TagSuffix:          "bar",
				TagSuffixSeparator: "bar",
			},
		},
		{
			r: Rule{
				Path: `{{define "foo"}} FOO `,
			},
			result: Rule{
				Path: `{{define "foo"}} FOO `,
			},
		},
		{
			r: Rule{
				Path: "{{ .foo }}",
				Tag:  `{{define "foo"}} FOO `,
			},
			result: Rule{
				Path: "bar",
				Tag:  `{{define "foo"}} FOO `,
			},
		},
		{
			r: Rule{
				Path:      "{{ .foo }}",
				Tag:       "{{ .foo }}",
				TagSuffix: `{{define "foo"}} FOO `,
			},
			result: Rule{
				Path:      "bar",
				Tag:       "bar",
				TagSuffix: `{{define "foo"}} FOO `,
			},
		},
		{
			r: Rule{
				Path:               "{{ .foo }}",
				Tag:                "{{ .foo }}",
				TagSuffix:          "{{ .foo }}",
				TagSuffixSeparator: `{{define "foo"}} FOO `,
			},
			result: Rule{
				Path:               "bar",
				Tag:                "bar",
				TagSuffix:          "bar",
				TagSuffixSeparator: `{{define "foo"}} FOO `,
			},
		},
	}
	for _, test := range tests {
		test.r.ParseAsTemplate(map[string]string{
			"foo": "bar",
		})
		if !reflect.DeepEqual(test.r, test.result) {
			t.Errorf("Rule %v not equal to %v", test.r, test.result)
		}
	}
}
