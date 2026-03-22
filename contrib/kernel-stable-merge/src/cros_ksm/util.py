# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
import os
import shlex
import sys
from datetime import UTC, datetime
from typing import NoReturn

NOW = datetime.now(tz=UTC)

def assert_unreachable() -> NoReturn:
    """Hint for static analysis.

    See <https://typing.python.org/en/latest/guides/unreachable.html#marking-code-as-unreachable>.
    """
    assert False, "unreachable"

def eprint(*args, **kwargs):
    """Like `builtins.print` but defaults to `sys.stderr`."""
    print(*args, file=sys.stderr, **kwargs)

def die(message: str | Exception | None = None, /) -> NoReturn:
    if message is not None:
        eprint(message)
    sys.exit(1)

def env_get_bool(name: str, /, default: bool) -> bool:
    value = os.environ.get(name, "")
    match value.lower():
        case "":
            return default
        case "1" | "true" | "on":
            return True
        case "0" | "false" | "off":
            return False
        case _:
            eprint(f"warning: ignoring env var {name}={value!r}")
            return default

def chomp[T: (str, bytes)](x: T, /) -> T:
    """Remove the record separator at the end of the string.

    Remove exactly one "record separator" from the end of the string, if it exists.
    The record separator is the longest of "\\n", "\\r", or "\\r\\n".

    Similar to `chomp` in Perl and Ruby.
    """
    if isinstance(x, str):
        tails = ("\r\n", "\n", "\r")
    elif isinstance(x, bytes):
        tails = (b"\r\n", b"\n", b"\r")
    else:
        raise TypeError(type(x))

    for tail in tails:
        n = len(tail)
        if x[-n:] == tail:
            return x[:-n]

    return x

def shquote_args(args: list) -> str:
    "Convert to a space-separated list of shell-quoted args."""
    return " ".join(map(shlex.quote, map(str, args)))

def args_add_if(args: list, cond: bool, yes: str, no: str | None = None, /) -> None:
    """Append string to an arg list if a condition is true.

    Append the `yes` string if `cond` is true. Otherwise append the `no` string
    if it is set.
    """
    if cond:
        args.append(yes)
    elif no is not None:
        args.append(no)
    else:
        pass

def datetime_is_naive(d: datetime, /) -> bool:
    """Test if datetime has naive timezone.

    See Also:
        - https://docs.python.org/3/library/datetime.html#aware-and-naive-objects
        - https://docs.python.org/3/library/datetime.html#determining-if-an-object-is-aware-or-naive
    """
    return (d.tzinfo is None) or (d.tzinfo.utcoffset(d) is None)

def format_utc(d: datetime, /) -> str:
    """Convert to UTC and return ISO 8601 format."""
    if datetime_is_naive(d):
        raise ValueError("datetime is naive")
    return format(d.astimezone(UTC), "%Y-%m-%dT%H:%M:%SZ")

DEBUG = env_get_bool("CROS_KSM_DEBUG", False)

def _init_log():
    if DEBUG:
        level = logging.DEBUG
    else:
        level = None

    logging.basicConfig(
        level = level,
        stream = sys.stderr,
        style = "{",
        format = "{levelname}: {message}",
    )

    return logging.getLogger()

log = _init_log()
