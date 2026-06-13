# Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/lang/pt-BR/) and uses [Conventional Commits](https://www.conventionalcommits.org/pt-br/v1.0.0/) for automatic changelog generation via [GoReleaser](https://goreleaser.com/).

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Cross-platform release pipeline for Linux, macOS and Windows.
- Automated GitHub Releases with `goreleaser`.
- `jira version` command showing build version, commit and date.

## How we version

1. New work is merged to `main` following Conventional Commits (e.g. `feat:`, `fix:`).
2. When a release is desired, create a git tag following SemVer:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. The `Release` GitHub Actions workflow builds the binaries for all platforms, generates the changelog from commits and publishes everything to the GitHub Release page.
4. After the release, the changelog section for that version is appended to this file manually (or via a follow-up PR) for a permanent, human-readable history.
