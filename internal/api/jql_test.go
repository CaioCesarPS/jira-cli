package api

import (
	"strings"
	"testing"
	"time"
)

func TestJQLBuilder_ProjectSeed(t *testing.T) {
	tests := []struct {
		name    string
		project string
		want    string
	}{
		{"with project", "SED", `project = "SED" ORDER BY created DESC`},
		{"empty project", "", `* ORDER BY created DESC`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewJQL(tt.project).Build()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJQLBuilder_Filters(t *testing.T) {
	user := &User{AccountID: "5f-abc"}
	tests := []struct {
		name string
		build func() string
		want string
	}{
		{
			"type only",
			func() string { return NewJQL("SED").Type("Bug").Build() },
			`project = "SED" AND type = "Bug" ORDER BY created DESC`,
		},
		{
			"single status",
			func() string { return NewJQL("").Status("To Do").Build() },
			`status IN ("To Do") ORDER BY created DESC`,
		},
		{
			"multiple statuses",
			func() string { return NewJQL("").Status("To Do", "In Progress").Build() },
			`status IN ("To Do", "In Progress") ORDER BY created DESC`,
		},
		{
			"empty status slice is no-op",
			func() string { return NewJQL("").Status("", "  ").Build() },
			`* ORDER BY created DESC`,
		},
		{
			"priority",
			func() string { return NewJQL("").Priority("High").Build() },
			`priority = "High" ORDER BY created DESC`,
		},
		{
			"assignee me with user",
			func() string { return NewJQL("").Assignee("me", user).Build() },
			`assignee = "5f-abc" ORDER BY created DESC`,
		},
		{
			"assignee me without user is a no-op",
			func() string { return NewJQL("").Assignee("me", nil).Build() },
			`* ORDER BY created DESC`,
		},
		{
			"assignee unassigned",
			func() string { return NewJQL("").Assignee("x", user).Build() },
			`assignee IS EMPTY ORDER BY created DESC`,
		},
		{
			"assignee explicit account id",
			func() string { return NewJQL("").Assignee("5f-zzz", user).Build() },
			`assignee = "5f-zzz" ORDER BY created DESC`,
		},
		{
			"multiple labels",
			func() string { return NewJQL("").Labels("backend", "urgent").Build() },
			`labels = "backend" AND labels = "urgent" ORDER BY created DESC`,
		},
		{
			"component",
			func() string { return NewJQL("").Component("api").Build() },
			`component = "api" ORDER BY created DESC`,
		},
		{
			"text",
			func() string { return NewJQL("").Text("OAuth login").Build() },
			`text ~ "OAuth login" ORDER BY created DESC`,
		},
		{
			"parent",
			func() string { return NewJQL("").Parent("SED-100").Build() },
			`parent = "SED-100" ORDER BY created DESC`,
		},
		{
			"chained filters",
			func() string {
				return NewJQL("SED").
					Type("Bug").
					Status("To Do").
					Priority("High").
					Assignee("me", user).
					Labels("backend").
					Build()
			},
			`project = "SED" AND type = "Bug" AND status IN ("To Do") AND priority = "High" AND assignee = "5f-abc" AND labels = "backend" ORDER BY created DESC`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.build()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJQLBuilder_Dates(t *testing.T) {
	tests := []struct {
		name             string
		rel, after, before string
	}{
		{
			"created with relative",
			"week", "", "",
		},
		{
			"created after overrides rel",
			"week", "2026-01-01", "",
		},
		{
			"created before",
			"", "", "2026-12-31",
		},
		{
			"all empty is no-op",
			"", "", "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewJQL("").Created(tt.rel, tt.after, tt.before).Build()
			switch tt.name {
			case "created with relative":
				want := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
				if !strings.Contains(got, `created >= "`+want+`"`) {
					t.Errorf("got %q, expected to contain created >= %q", got, want)
				}
			case "created after overrides rel":
				if !strings.Contains(got, `created >= "2026-01-01"`) {
					t.Errorf("got %q, expected to use 'after' value", got)
				}
			case "created before":
				if !strings.Contains(got, `created <= "2026-12-31"`) {
					t.Errorf("got %q, expected created <= \"2026-12-31\"", got)
				}
			case "all empty is no-op":
				if strings.Contains(got, "created >= ") || strings.Contains(got, "created <= ") {
					t.Errorf("got %q, expected no created filter", got)
				}
			}
		})
	}
}

func TestJQLBuilder_OrderBy(t *testing.T) {
	tests := []struct {
		field   string
		reverse bool
		want    string
	}{
		{"created", false, "ORDER BY created DESC"},
		{"updated", false, "ORDER BY updated DESC"},
		{"priority", true, "ORDER BY priority ASC"},
	}
	for _, tt := range tests {
		got := NewJQL("").OrderBy(tt.field).Reverse(tt.reverse).Build()
		if !strings.Contains(got, tt.want) {
			t.Errorf("OrderBy(%q, %v) → %q missing %q", tt.field, tt.reverse, got, tt.want)
		}
	}
}

func TestResolveRelativeDate(t *testing.T) {
	tests := []struct {
		token  string
		expect string
	}{
		{"today", time.Now().Format("2006-01-02")},
		{"yesterday", time.Now().AddDate(0, 0, -1).Format("2006-01-02")},
		{"week", time.Now().AddDate(0, 0, -7).Format("2006-01-02")},
		{"month", time.Now().AddDate(0, -1, 0).Format("2006-01-02")},
		{"year", time.Now().AddDate(-1, 0, 0).Format("2006-01-02")},
		{"-3d", time.Now().AddDate(0, 0, -3).Format("2006-01-02")},
		{"-2w", time.Now().AddDate(0, 0, -14).Format("2006-01-02")},
		{"2026-06-10", "2026-06-10"},
		{"2026/06/10", "2026-06-10"},
		{"unknown-token", "unknown-token"}, // passed through unchanged
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := ResolveRelativeDate(tt.token)
			if got != tt.expect {
				t.Errorf("ResolveRelativeDate(%q) = %q, want %q", tt.token, got, tt.expect)
			}
		})
	}
}

func TestNonEmpty(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{[]string{"a", "", "b"}, []string{"a", "b"}},
		{[]string{"", "  "}, []string{}},
		{[]string{"x"}, []string{"x"}},
		{nil, []string{}},
	}
	for _, tt := range tests {
		got := nonEmpty(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("nonEmpty(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
