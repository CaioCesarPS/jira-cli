// Package version expõe informações de build da CLI.
// Os valores são injetados via ldflags no momento da compilação (make, goreleaser, etc.).
package version

import "fmt"

var (
	// Version é a versão semântica da release (ex: v1.2.3).
	Version = "dev"

	// Commit é o hash curto do commit de build.
	Commit = "unknown"

	// Date é a data/hora UTC do build no formato RFC3339.
	Date = "unknown"
)

// Info retorna uma string formatada com versão, commit e data de build.
func Info() string {
	return fmt.Sprintf("jira-cli version %s (commit: %s, built: %s)", Version, Commit, Date)
}
