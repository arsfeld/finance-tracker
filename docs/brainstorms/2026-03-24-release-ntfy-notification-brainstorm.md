# Brainstorm: Release Notes & Ntfy Notifications

**Date:** 2026-03-24
**Status:** Complete

## What We're Building

Two improvements to the release pipeline:

1. **Better release notes** — Replace bare `--generate-notes` (which only lists PRs, not direct commits) with **git-cliff**, which parses existing conventional commit messages into categorized changelogs (Features, Bug Fixes, etc.).

2. **Ntfy release notification** — After each release, send a notification to `ntfy.arsfeld.one/finance-tracker` with the changelog and a link to `finance-tracker.arsfeld.one`.

## Why This Approach

- **git-cliff** works with direct commits to main (unlike GitHub's `--generate-notes` which only sees merged PRs). Our commit messages already follow conventional format (`feat(web):`, `fix(categories):`, `docs:`, etc.) so it works out of the box.
- **GitHub Actions step** — keeps all release automation in `.github/workflows/release.yaml` with zero Go code changes.
- **Repository variables** for ntfy config — easy to change without editing workflow YAML.

## Key Decisions

1. **Release notes tool:** git-cliff with `orhun/git-cliff-action@v4`. Add a `cliff.toml` config to the repo root.
2. **Trigger location:** Two new steps in the `create-release` job — git-cliff before `gh release create`, ntfy notification after.
3. **Replace `--generate-notes`:** Use `--notes` with git-cliff output instead.
4. **Message format:** Version as ntfy title, categorized changelog as body, click-through link to `finance-tracker.arsfeld.one`.
5. **Ntfy config:** Store as GitHub repo variables (`vars.NTFY_RELEASE_SERVER`, `vars.NTFY_RELEASE_TOPIC`).
6. **Failure handling:** ntfy step uses `continue-on-error: true` so notification failures don't block releases.

## Implementation Sketch

### cliff.toml (repo root)

```toml
[changelog]
body = """
{% for group, commits in commits | group_by(attribute="group") %}
### {{ group | upper_first }}
{% for commit in commits %}
- {% if commit.scope %}**{{ commit.scope }}**: {% endif %}{{ commit.message | split(pat="\n") | first | upper_first | trim }}
{% endfor %}
{% endfor %}
"""
trim = true

[git]
conventional_commits = true
filter_unconventional = false
commit_parsers = [
    { message = "^feat", group = "Features" },
    { message = "^fix", group = "Bug Fixes" },
    { message = "^doc", group = "Documentation" },
    { message = "^perf", group = "Performance" },
    { message = "^refactor", group = "Refactoring" },
    { message = "^style", group = "Styling" },
    { message = "^test", group = "Testing" },
    { message = "^chore", group = "Miscellaneous" },
    { message = "^ci", group = "CI/CD" },
    { message = "^.*", group = "Other" },
]
filter_commits = false
tag_pattern = "v[0-9].*"
sort_commits = "oldest"
```

### Workflow changes (create-release job)

```yaml
      - name: Checkout the code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required for git-cliff

      - name: Generate Release Notes
        id: changelog
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --verbose --latest --strip header

      - name: Create Release
        env:
          GH_TOKEN: ${{ github.token }}
        run: |
          new_tag="${{ needs.tag-release.outputs.new_tag }}"
          gh release create "$new_tag" \
            --title "$new_tag" \
            --notes "${{ steps.changelog.outputs.content }}" \
            "release-bins/final/finance-tracker-linux-x86_64" \
            "release-bins/final/finance-tracker-linux-arm64"

      - name: Notify via ntfy
        continue-on-error: true
        run: |
          new_tag="${{ needs.tag-release.outputs.new_tag }}"
          curl -s \
            -H "Title: Finance Tracker ${new_tag} released" \
            -H "Tags: rocket" \
            -H "Click: https://finance-tracker.arsfeld.one" \
            -d "${{ steps.changelog.outputs.content }}

View app: https://finance-tracker.arsfeld.one" \
            "https://${{ vars.NTFY_RELEASE_SERVER }}/${{ vars.NTFY_RELEASE_TOPIC }}"
```

## GitHub Repo Variables Needed

- `NTFY_RELEASE_SERVER` = `ntfy.arsfeld.one`
- `NTFY_RELEASE_TOPIC` = `finance-tracker`
