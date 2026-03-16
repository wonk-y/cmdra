---
sidebar_position: 3
---

# PTY Attach Checklist

Use this checklist when validating PTY-backed attach behavior in `cmdraui` or `cmdractl`.

## Core shell checks

1. Start a PTY-backed shell session.
2. Attach to it.
3. Verify the shell prompt appears immediately.
4. Type a short command and confirm normal echo:
   - `echo hello`
5. Test line editing:
   - type a long command
   - use left/right arrows
   - use backspace
   - use `ctrl+a` and `ctrl+e`
6. Test history navigation:
   - run two commands
   - use up/down arrows
7. Test a clear/redraw:
   - Unix: `clear`
   - Windows: `cls`
8. Resize the local terminal smaller and larger.
9. Verify the remote PTY size updates:
   - Unix: `stty size`
   - Windows PowerShell: `$Host.UI.RawUI.WindowSize`

## Attach lifecycle checks

1. Detach with `ctrl+g q`.
2. Reattach and confirm the session is still usable.
3. Cancel with `ctrl+g c`.
4. Confirm the attached process exits and `cmdraui` returns to the normal 3-pane view.
5. Start another session and let the shell exit normally.
6. Confirm `cmdraui` keeps the exited attach view visible until you detach explicitly.

## PTY app checks

These are not expected to be perfect yet. They help identify the next missing terminal behaviors.

1. Unix:
   - `less README.md`
   - `vim README.md`
   - `top` or `htop`
2. Windows:
   - `more README.md`
   - `powershell`
   - `cmd`

Record whether the issue is primarily:

- input handling
- cursor motion
- erase/redraw
- wrapping
- resize
- alternate screen
- color/style

## Comparison check

If a PTY issue appears in `cmdraui`, repeat the same session with `cmdractl attach`.

- If it reproduces in both, the issue is likely in the server/PTTY path.
- If it only reproduces in `cmdraui`, the issue is likely in the TUI emulator path.
