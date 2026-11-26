# Agent Instructions

## Issue tracking

Work for the Interfacer ecosystem is tracked in GitHub issues on
[interfacerproject/interfacer-gui](https://github.com/interfacerproject/interfacer-gui/issues)
(`gh issue list -R interfacerproject/interfacer-gui`). Epics are parent issues with
sub-issues; blockers use GitHub issue dependencies. Do not keep markdown TODO files as
a parallel tracker.

Never file security problems as public issues: report them through a private GitHub
security advisory.

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**
- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var
