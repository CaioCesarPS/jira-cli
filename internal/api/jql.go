package api

import (
	"fmt"
	"strings"
	"time"
)

// JQLBuilder constructs a Jira Query Language (JQL) string incrementally.
// Each setter returns the same builder to allow chaining:
//
//	api.NewJQL("SED").Type("Bug").Status("To Do").Assignee("me", &user).Build()
type JQLBuilder struct {
	conditions []string
	orderBy    string
	reverse    bool
}

// NewJQL creates a JQLBuilder. If project is non-empty, it seeds the query
// with a `project = "<key>"` condition.
func NewJQL(project string) *JQLBuilder {
	j := &JQLBuilder{orderBy: "created"}
	if project != "" {
		j.conditions = append(j.conditions, fmt.Sprintf("project = %q", project))
	}
	return j
}

// Type filters by issue type (e.g. "Task", "Bug", "Story").
func (j *JQLBuilder) Type(t string) *JQLBuilder {
	if t == "" {
		return j
	}
	j.conditions = append(j.conditions, fmt.Sprintf("type = %q", t))
	return j
}

// Status filters by one or more statuses. Multiple are combined with IN.
func (j *JQLBuilder) Status(statuses ...string) *JQLBuilder {
	statuses = nonEmpty(statuses)
	if len(statuses) == 0 {
		return j
	}
	quoted := make([]string, len(statuses))
	for i, s := range statuses {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	j.conditions = append(j.conditions, fmt.Sprintf("status IN (%s)", strings.Join(quoted, ", ")))
	return j
}

// Priority filters by issue priority (e.g. "High", "Medium", "Low").
func (j *JQLBuilder) Priority(p string) *JQLBuilder {
	if p == "" {
		return j
	}
	j.conditions = append(j.conditions, fmt.Sprintf("priority = %q", p))
	return j
}

// Assignee filters by assignee.
//
// Special values:
//   - "me"     → uses currentUser.AccountID (must not be nil)
//   - "x", "none", "unassigned" → matches issues with no assignee
//   - any other string → matched literally as an account id
//
// An empty value is treated as "me" if currentUser is non-nil, otherwise it
// is a no-op (we never want to silently generate a bad query).
func (j *JQLBuilder) Assignee(value string, currentUser *User) *JQLBuilder {
	switch value {
	case "me":
		if currentUser == nil {
			return j // skip — caller did not resolve current user
		}
		j.conditions = append(j.conditions, fmt.Sprintf("assignee = %q", currentUser.AccountID))
	case "x", "none", "unassigned":
		j.conditions = append(j.conditions, "assignee IS EMPTY")
	case "":
		if currentUser != nil {
			j.conditions = append(j.conditions, fmt.Sprintf("assignee = %q", currentUser.AccountID))
		}
	default:
		j.conditions = append(j.conditions, fmt.Sprintf("assignee = %q", value))
	}
	return j
}

// Reporter filters by reporter. Same special values as Assignee.
func (j *JQLBuilder) Reporter(value string, currentUser *User) *JQLBuilder {
	switch value {
	case "me":
		if currentUser == nil {
			return j
		}
		j.conditions = append(j.conditions, fmt.Sprintf("reporter = %q", currentUser.AccountID))
	case "":
		if currentUser != nil {
			j.conditions = append(j.conditions, fmt.Sprintf("reporter = %q", currentUser.AccountID))
		}
	default:
		j.conditions = append(j.conditions, fmt.Sprintf("reporter = %q", value))
	}
	return j
}

// Labels adds one or more label filters (combined with AND).
func (j *JQLBuilder) Labels(labels ...string) *JQLBuilder {
	for _, l := range nonEmpty(labels) {
		j.conditions = append(j.conditions, fmt.Sprintf("labels = %q", l))
	}
	return j
}

// Component filters by component name.
func (j *JQLBuilder) Component(c string) *JQLBuilder {
	if c == "" {
		return j
	}
	j.conditions = append(j.conditions, fmt.Sprintf("component = %q", c))
	return j
}

// Text adds a full-text search condition.
func (j *JQLBuilder) Text(text string) *JQLBuilder {
	if text == "" {
		return j
	}
	j.conditions = append(j.conditions, fmt.Sprintf("text ~ %q", text))
	return j
}

// Parent filters by parent issue key.
func (j *JQLBuilder) Parent(parentKey string) *JQLBuilder {
	if parentKey == "" {
		return j
	}
	j.conditions = append(j.conditions, fmt.Sprintf("parent = %q", parentKey))
	return j
}

// Created adds a `created >=` filter. `rel` accepts either a relative token
// resolved by ResolveRelativeDate (e.g. "today", "week", "-10d") or an
// absolute date in yyyy-mm-dd / yyyy/mm/dd format. `after` and `before`
// take absolute dates; if either is set it overrides `rel`.
func (j *JQLBuilder) Created(rel, after, before string) *JQLBuilder {
	date, used := resolveDateOrEmpty(rel, after, before)
	if date == "" {
		return j
	}
	if used == "before" {
		j.conditions = append(j.conditions, fmt.Sprintf("created <= %q", date))
	} else {
		j.conditions = append(j.conditions, fmt.Sprintf("created >= %q", date))
	}
	return j
}

// Updated mirrors Created but for the `updated` field.
func (j *JQLBuilder) Updated(rel, after, before string) *JQLBuilder {
	date, used := resolveDateOrEmpty(rel, after, before)
	if date == "" {
		return j
	}
	if used == "before" {
		j.conditions = append(j.conditions, fmt.Sprintf("updated <= %q", date))
	} else {
		j.conditions = append(j.conditions, fmt.Sprintf("updated >= %q", date))
	}
	return j
}

// OrderBy sets the field used for ORDER BY. Default is "created".
func (j *JQLBuilder) OrderBy(field string) *JQLBuilder {
	if field == "" {
		return j
	}
	j.orderBy = field
	return j
}

// Reverse sets the builder to ascending sort order. Pass true to flip
// the default (which is DESC).
func (j *JQLBuilder) Reverse(asc bool) *JQLBuilder {
	j.reverse = asc
	return j
}

// Build returns the final JQL string. If no conditions were set, returns "*"
// (match everything). Always appends ORDER BY unless explicitly empty.
func (j *JQLBuilder) Build() string {
	var where string
	if len(j.conditions) == 0 {
		where = "*"
	} else {
		where = strings.Join(j.conditions, " AND ")
	}

	direction := "DESC"
	if j.reverse {
		direction = "ASC"
	}

	if j.orderBy == "" {
		return where
	}
	return fmt.Sprintf("%s ORDER BY %s %s", where, j.orderBy, direction)
}

// ── helpers ────────────────────────────────────────────────────────────────

func nonEmpty(s []string) []string {
	out := make([]string, 0, len(s))
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

// resolveDateOrEmpty normalises the (rel, after, before) triple into a
// single date string and a flag indicating which one was used.
// Returns ("", "") if nothing usable was provided.
func resolveDateOrEmpty(rel, after, before string) (string, string) {
	switch {
	case after != "":
		return normaliseDate(after), "after"
	case before != "":
		return normaliseDate(before), "before"
	case rel != "":
		return ResolveRelativeDate(rel), "after"
	}
	return "", ""
}

// normaliseDate converts "yyyy/mm/dd" to "yyyy-mm-dd" (Jira expects dashes).
// If parsing fails, returns the input unchanged so the user gets a meaningful
// API error rather than a silent rewrite.
func normaliseDate(s string) string {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006/01/02", s); err == nil {
		return t.Format("2006-01-02")
	}
	return s
}

// ResolveRelativeDate converts a relative date token to an absolute
// yyyy-mm-dd string. Supported tokens:
//
//	"today", "yesterday"
//	"week", "month", "year"        → start of that period
//	"-Nd", "-Nw", "-Nh", "-Nm"     → now minus N days/weeks/hours/minutes
//	"yyyy-mm-dd" or "yyyy/mm/dd"   → returned (normalised) as-is
//
// Any unrecognised input is returned unchanged so the user sees the
// error from the Jira API rather than us guessing.
func ResolveRelativeDate(token string) string {
	t := strings.ToLower(strings.TrimSpace(token))
	now := time.Now()

	switch t {
	case "today":
		return now.Format("2006-01-02")
	case "yesterday":
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case "week":
		return now.AddDate(0, 0, -7).Format("2006-01-02")
	case "month":
		return now.AddDate(0, -1, 0).Format("2006-01-02")
	case "year":
		return now.AddDate(-1, 0, 0).Format("2006-01-02")
	}

	// Period format: -Nd / -Nw / -Nh / -Nm
	if len(t) >= 3 && t[0] == '-' {
		numStr := t[1 : len(t)-1]
		unit := t[len(t)-1]
		var n int
		if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil {
			switch unit {
			case 'd':
				return now.AddDate(0, 0, -n).Format("2006-01-02")
			case 'w':
				return now.AddDate(0, 0, -7*n).Format("2006-01-02")
			case 'h':
				return now.Add(-time.Duration(n) * time.Hour).Format("2006-01-02")
			case 'm':
				return now.Add(-time.Duration(n) * time.Minute).Format("2006-01-02")
			}
		}
	}

	// Absolute date — normalise
	return normaliseDate(token)
}
