# Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/lang/pt-BR/) and uses [Conventional Commits](https://www.conventionalcommits.org/pt-br/v1.0.0/) for automatic changelog generation via [GoReleaser](https://goreleaser.com/).

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Cross-platform release pipeline for Linux, macOS and Windows.
- Automated GitHub Releases with `goreleaser`.
- `jira version` command showing build version, commit and date.
- `install.sh` (Linux/macOS/Git Bash) and `install.ps1` (Windows PowerShell) installers.

## How we version

1. New work is merged to `main` following Conventional Commits (e.g. `feat:`, `fix:`).
2. On every successful push to `main`, the `Auto-release` job (part of the `CI` workflow)
   inspects commits since the last tag. If at least one `feat:`, `fix:`, `perf:`, `refactor:`,
   or `BREAKING CHANGE` commit is found, it computes the next SemVer tag (major/minor/patch),
   creates and pushes it automatically. Commits that are only `docs`/`style`/`test`/`chore`/`ci`/`build`
   do not trigger a release.
3. The same job then runs GoReleaser to build the binaries for all platforms, generate the
   changelog from commits, and publish everything to the GitHub Release page.
4. A release can still be cut manually by pushing a tag by hand — the `Release` workflow
   (triggered on any `v*` tag push) builds and publishes it the same way:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
5. After the release, the changelog section for that version is appended to this file manually (or via a follow-up PR) for a permanent, human-readable history.
