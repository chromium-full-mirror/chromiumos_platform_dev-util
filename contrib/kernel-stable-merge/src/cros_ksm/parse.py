# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Parsing for commit messages."""

# Be loose with type checking because parsy makes it difficult. The parsy
# maintainer wrote an article[1] about it.
#
# [1]: https://lukeplant.me.uk/blog/posts/python-type-hints-parsy-case-study/

from __future__ import annotations

import enum
from dataclasses import dataclass
from typing import Self

import parsy
from parsy import (
    Parser,
    Result,
    char_from,
    eof,
    peek,
    regex,
    string,
    string_from,
)


def _check_same_streams(a, b):
    # This check is too strict. We only require that the two streams have the
    # same storage and start location. The length of the stream is
    # insignificant.
    if a is not b:
        raise ValueError("different streams")

def _check_range(a, b):
    if a > b:
        raise ValueError("negative range")

@dataclass(frozen=True)
class SourceLocation:

    stream: str
    index: int

    def __sub__(self, other: Self) -> SourceRange:
        assert isinstance(other, SourceLocation)
        _check_same_streams(self.stream, other.stream)
        return SourceRange(self.stream, other.index, self.index)

    @property
    def line_info(self) -> str:
        line, col = parsy.line_info_at(self.stream, self.index)
        return f"{line}:{col}"

@dataclass(frozen=True)
class SourceRange:

    stream: str
    start_index: int
    stop_index: int

    def __post_init__(self):
        _check_range(self.start_index, self.stop_index)

    def __len__(self):
        return self.stop_index - self.start_index

    @property
    def start_loc(self):
        return SourceLocation(self.stream, self.start_index)

    @property
    def stop_loc(self):
        return SourceLocation(self.stream, self.stop_index)

    @property
    def text(self) -> str:
        return self.stream[self.start_index:self.stop_index]

@dataclass(frozen=True)
class ClumsyString:
    """A string delimited with opening/closing quotes or brackets.

    This string type differs from the usual string literals found in programming
    languages. Its parsing rules are clumsy. The string's inner text can contain
    anything, including the string's outer delimiters in unescaped form.

    For example, each line below is a valid ClumsyString, but may not be a valid
    string literal under typical string parsing rules.

        "hello"
        "hello "bye""
        "hello "bye"
        (hello)
        (hello ( bye)
        (hello ) bye)

    """
    source: SourceRange
    inner_source: SourceRange
    left_quote: str
    right_quote: str

class SubjectInnerType(enum.Enum):
    NONE = enum.auto()
    REVERT = enum.auto()
    PORT = enum.auto()

@dataclass(frozen=True)
class Subject:
    source: SourceRange
    inner_subject: Self | None
    inner_type: SubjectInnerType

@dataclass
class Trailer:
    source: SourceRange
    key: str
    value: str
    oid: str | None = None
    subject: Subject | None = None

    def __post_init__(self):
        assert self.key.lower() in PARSED_TRAILER_KEYS

@Parser
def get_source_loc(stream, index) -> Result:
    """A parser that produces the current SourceLocation."""
    value = SourceLocation(stream, index)
    return Result.success(index, value)

def jump_index(index: int, /) -> Parser:
    """A parser that jumps to the stream index."""

    @Parser
    def impl(stream, _index) -> Result:
        return Result.success(index, None)

    return impl

eol = string("\n")
space = regex(r"\s+")
pad = regex(r"\s*")
colon = char_from(":")

until_eol = regex(r".*?\n")
until_eof = regex(r".*") << eof

oid = regex(r"\b[0-9A-Za-z]{8,40}\b")
subject_any_prefix = regex(r"[a-zA-Z0-9/_-]+:") + space
subject_port_prefix = (
    string_from(
        "CHROMIUM",
        "BACKPORT",
        "FORMGIT", # typo
        "FROMGIT",
        "FORMLIST", # typo
        "FROMLIST",
        "UPSTREAM",
    )
    + char_from(":")
    + space
)
subject_revert_prefix = string_from("Revert") + space

def up_to_index(stop_index: int, parser: Parser, /) -> Parser:
    """Wrap the parser so that it does not the parse the stream past the given
    location.
    """

    @Parser
    def impl(stream, index) -> Result:
        _check_range(index, stop_index)
        return parser(stream[:stop_index], index)

    return impl

@Parser
def clumsy_string(stream, index) -> Result:
    """A parser that produces a ClumsyString.

    This is a greedy parser that parses both left-to-right and right-to-left.

    The stream's current index must be an opening quote or bracket. The parser
    searches right-to-left, starting at the end of the stream, for the matching
    closing character. On success, the stream's index is set to the index after
    the closing character.

    Use `up_to_index` to control the starting point of the right-to-left search.

    This behavior intentionally differs from normal string parsing.  Due to the
    right-to-left parsing, the parser's behavior is unaffected by the inner
    text between the opening and closing characters.
    """
    assert isinstance(stream, str)

    start_index = index

    left_quote_chars = "\"'(["
    right_quote_chars = "\"')]"

    left_quote = stream[index]
    j = left_quote_chars.find(left_quote)
    if j == -1:
        return Result.failure(index, f"not a left quote char: {left_quote}!r")

    right_quote = right_quote_chars[j]

    # ruff: ignore[redefined-argument-from-local]
    for index in range(len(stream) - 1, start_index, -1):
        if stream[index] == right_quote:
            value = ClumsyString(
                source = SourceRange(stream, start_index, index + 1),
                inner_source = SourceRange(stream, start_index + 1, index),
                left_quote = left_quote,
                right_quote = right_quote,
            )
            return Result.success(index + 1, value)

    return Result.failure(start_index,
                          f"opening {left_quote} has no closing {right_quote}")

@parsy.generate
def subject_with_inner_port():
    start_loc = yield get_source_loc
    yield until_eol | until_eof
    stop_loc = yield get_source_loc

    yield jump_index(start_loc.index)
    yield subject_port_prefix.at_least(1)
    inner_subject = yield up_to_index(stop_loc.index, subject)
    return Subject(
        source = stop_loc - start_loc,
        inner_subject = inner_subject,
        inner_type = SubjectInnerType.PORT,
    )

@parsy.generate
def subject_with_inner_revert():
    start_loc = yield get_source_loc
    yield until_eol | until_eof
    stop_loc = yield get_source_loc

    yield jump_index(start_loc.index)
    yield subject_any_prefix.many()
    yield subject_revert_prefix
    cs = yield up_to_index(stop_loc.index, clumsy_string)
    if cs.source.stop_loc != stop_loc:
        yield parsy.fail("trailing text appears after quoted reverted subject")

    yield jump_index(cs.inner_source.start_loc.index)
    inner_subject = yield up_to_index(cs.inner_source.stop_index, subject)

    yield jump_index(stop_loc.index)
    return Subject(
        source = stop_loc - start_loc,
        inner_subject = inner_subject,
        inner_type = SubjectInnerType.REVERT,
    )

@parsy.generate
def subject():
    x = yield subject_with_inner_port.optional()
    if x:
        return x

    x = yield subject_with_inner_revert.optional()
    if x:
        return x

    start_loc = yield get_source_loc
    yield until_eol | until_eof
    stop_loc = yield get_source_loc
    return Subject(
        source = stop_loc - start_loc,
        inner_subject = None,
        inner_type = SubjectInnerType.NONE,
    )

# We parse only these trailer keys.
PARSED_TRAILER_KEYS = {
    "fixes",
    "stable-dep-of",
}

trailer_key = regex("(?i)" + "|".join(PARSED_TRAILER_KEYS))

@parsy.generate
def trailer_subject():
    """
    Parse a bracketed subject that follows an oid in a trailer value. Discard
    any non-subject info inside the brackets.

    The most common form is ```("subject")```, but there also exist unusual
    forms in the kernel history.

    Example Forms:

        ("subject")
        ('subject')
        (subject)
        "subject"
        'subject'

        ("subject", extra)
        ("subject" extra)
        ("subject") extra
        "subject" extra
    """

    start_loc = yield get_source_loc
    yield until_eol | until_eof
    stop_loc = yield get_source_loc

    # Parse the ClumsyString, but place an upper bound on the subparser to prevent
    # it from continuing past EOL into the next trailer. The `clumsy_string` parser
    # is greedy.
    yield jump_index(start_loc.index)
    cs = yield up_to_index(stop_loc.index, clumsy_string)

    if cs.left_quote in {"(", "["}:
        yield jump_index(cs.inner_source.start_index)
        yield pad
        inner_left_quote = yield peek(string_from('"', "'")).optional()
        if inner_left_quote:
            # Drill down into the inner quote
            cs = yield up_to_index(cs.inner_source.stop_index, clumsy_string)

    yield jump_index(cs.inner_source.start_index)
    s = yield up_to_index(cs.inner_source.stop_index, subject)

    # Discard any extra info inside the outermost quote
    yield jump_index(stop_loc.index)

    return s

@parsy.generate
def trailer():
    start_loc = yield get_source_loc
    key = yield trailer_key
    yield colon >> pad
    value_loc = yield get_source_loc
    value = yield until_eol | until_eof
    stop_loc = yield get_source_loc

    t = Trailer(stop_loc - start_loc, key, value)

    # Rewind to parse the trailer value.
    yield jump_index(value_loc.index)

    # Parse optional oid.
    yield pad
    t.oid = yield oid.optional()
    if t.oid is not None:
        yield pad
        t.subject = yield trailer_subject.optional()

    # Ignore any trailing info
    yield jump_index(stop_loc.index)
    return t
