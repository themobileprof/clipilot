# Clio on Termux (Android)

Clio is a CLI pilot for navigating the Linux terminal—especially on mobile. Termux is a first-class platform.

## Install

```bash
# Install Termux from F-Droid or GitHub releases (not Play Store)
curl -fsSL https://clio.themobileprof.com/clio | sh
```

Close Termux and open it again, then:

```bash
clio
```

At the `>>` prompt:

```text
sync
setup termux
exit
```

In the shell (`$` prompt), run the wizard Clio printed, for example:

```bash
clio-run-module termux_setup setup
```

## Everyday use

```bash
clio                  # open the REPL
clio ask 'list files' # one-shot question (if supported by your build)
```

Inside the REPL:

- Plain English questions → shell commands (confirm before run)
- `setup` — setup wizards
- `modules` / `catalog` — browse automation
- `sync` / `sync full` — refresh from the registry

**Do not paste `clio-run-module …` at the `>>` prompt.** Exit Clio first, then run it in Termux’s normal shell.

## Tips

1. Prefer Zsh after `clio termux_setup` / related wizards
2. Use a lite sync on low-memory devices: `sync` first; `sync full` when you need everything
3. If you see `SIGSYS: bad system call`, you tried to run a module workflow inside the REPL — exit and use `clio-run-module` in the shell

## Registry

Modules and wizards come from **https://clio.themobileprof.com**.

## Uninstall

Follow the Clio uninstall steps in the [Clio repository](https://github.com/themobileprof/clio). Package installs you ran via wizards (git, python, etc.) stay until you remove them with `pkg uninstall`.
