# Agent Instructions

This project uses **OpenSpec** for change planning and task tracking.

## OpenSpec workflow

```bash
openspec list --json                 # List active changes
openspec status --change <name>      # Inspect artifact and task status
openspec validate <name>             # Validate a change
```

- Use `openspec/changes/<name>/tasks.md` as the source of truth for implementation progress.
- Use `/opsx:propose` to create a change, `/opsx:apply` to implement it, and `/opsx:archive` after completion.
- Keep proposal, design, capability specs, and tasks aligned as decisions change.
- Do not create a second task-tracking system alongside OpenSpec.

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

```bash
cp -f source dest
mv -f source dest
rm -f file
rm -rf directory
cp -rf source dest
```

Other commands that may prompt:

- `scp` — use `-o BatchMode=yes`
- `ssh` — use `-o BatchMode=yes`
- `apt-get` — use `-y`
- `brew` — use `HOMEBREW_NO_AUTO_UPDATE=1`

## Session completion

Work is not complete until the relevant OpenSpec tasks are updated and the Git changes are committed and pushed.

1. Capture remaining work in the active change's `tasks.md`.
2. Run quality gates when code changed.
3. Update completed and pending task checkboxes.
4. Run `openspec validate <change>` for affected changes.
5. Run `git pull --rebase`, `git push`, and `git status`.
6. Confirm the branch is up to date with its remote.
7. Hand off remaining context.
