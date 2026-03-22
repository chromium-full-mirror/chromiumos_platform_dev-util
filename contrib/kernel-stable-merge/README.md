# cros-ksm

A tool for managing ChromeOS kernel stable merges.

## Quickstart

Run `./bin/cros-ksm`. You can also add the `bin` directory to your shell's `PATH`.

## Dependencies

### [uv][]

This project uses [uv][] to manage its Python dependencies and virtualenv.  The
project provides multiple quick installation methods such as: via standalone
installer script, via `pipx`, via `cargo`, and more.
See [https://docs.astral.sh/uv/getting-started/installation/].

*Why uv?* It is similar to `pip` and `poetry`, but is not slow and is less
awkward to use. For example, `poetry run` suffered from noticeable startup delay
every time `cros-ksm` was run.  Also, when project's dependencies or structure
changed, `poetry` would often require the user to somehow detect the change and
then manually re-run `poetry install`. In comparison, `uv run` has negligible
startup time and automatically re-runs the install step when needed. Also, its
documentation is easier to navigate and understand.

[uv]: https://docs.astral.sh/uv

## Development

Install [just](https://just.systems/).
`sudo apt-dnf-pacman-whatever install just`.

Run `just check` before each commit.

Run `just --list` to view all just targets.

The unit tests require that environment variable `CROS_KSM_TEST_KERNEL_GIT_DIR`
be set to the gitdir of an up-to-date 6.12 kernel repo. If the tests fail due to
missing commits, you probably need to fetch the relevant remotes.

Example:
```
$ pushd $CROS_TOP/src/third_party/kernel/v6.12
$ cros-ksm fetch
$ export CROS_KSM_TEST_KERNEL_GIT_DIR="$PWD/.git"
$ popd
$ just check
```

### How to fix ruff errors

If the `check` script fails due to `ruff` errors, then try asking `ruff` to fix
them automatically. Run `just ruff-fix`.

### How to run pytest manually

Run `uv run pytest`. Useful flags are `-v|--verbose`, `-x|--exitfirst`,
`--runxfail`, and `--collect-only|--co`.

## Debugging

Set environment variable `CROS_KSM_DEBUG=1`.

## BUGS

Many known bugs are documented as expected test failures. To find them, run `git
grep xfail` and read the nearby comments. To inspect the failures, run `uv run
pytest -v --runxfail`.
