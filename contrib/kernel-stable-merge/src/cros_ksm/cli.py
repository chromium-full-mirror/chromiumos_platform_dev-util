# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

__doc__ = """
Helpers for writing a CLI.

This module's primary purpose is to override undesirable behavior in `click`.
"""

import shutil
from collections.abc import Iterable
from typing import override

import click


def wrap_text(
    text: str,
    width: int | None = None,
    initial_indent: str = "",
    subsequent_indent: str = "",
    preserve_paragraphs: bool = True,
) -> str:
    """Like `click.wrap_text`, but with different defaults.

    Changes defaults:
        - width: terminal width
        - preserve_paragraphs: true
    """
    if width is None:
        width = shutil.get_terminal_size().columns
    return click.wrap_text(
        text = text,
        width = width,
        initial_indent = initial_indent,
        subsequent_indent = subsequent_indent,
        preserve_paragraphs = preserve_paragraphs,
    )

class HelpFormatter(click.HelpFormatter):
    """Override `click.HelpFormatter`.

    Changes compared to click:
        - Default `max_width` is the terminal width, not 80ish.
        - Default `width` is terminal width, not 50.
        - Default indent increment is 4, not 2.
        - CLI options and their help text are formatted to resemble man pages.
          Click's default formatting uses a narrow 2-column output, with the
          help text wrapped to 30 columnts. The default formatting is intended
          only for short help text and is difficult to read if the option's help
          text is long, and is especially difficult if it contains multiple
          paragraphs or a pre-formatted list.
    """

    def __init__(
        self,
        indent_increment: int = 4,
        width: int | None = None,
        max_width: int | None = None,
    ):
        if max_width is None:
            max_width = shutil.get_terminal_size().columns

        if width is None:
            width = max_width

        super().__init__(
            indent_increment = indent_increment,
            width = width,
            max_width = max_width,
        )

    @property
    def current_indent_prefix(self) -> str:
        return " " * self.current_indent

    @override
    def write_dl(
        self,
        rows: Iterable[tuple[str, str]],
        col_max: int = 30, # ignored
        col_spacing: int = 2, # ignored
    ) -> None:
        prev_row_had_definition = False

        for term, definition in rows:
            if prev_row_had_definition:
                # Add empty line after the previous definition
                self.write("\n")

            self.write(f"{self.current_indent_prefix}{term}\n")

            if not definition:
                prev_row_had_definition = False
                continue

            with self.indentation():
                text = wrap_text(
                    definition, self.width,
                    initial_indent = self.current_indent_prefix,
                    subsequent_indent = self.current_indent_prefix,
                    preserve_paragraphs = True,
                )
                self.write(text)
                if not text.endswith("\n"):
                    self.write("\n")

            prev_row_had_definition = True

click.Context.formatter_class = HelpFormatter
