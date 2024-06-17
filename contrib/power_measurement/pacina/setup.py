# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Python package build metadata."""

import setuptools


# Read the contents of the README file
with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

# Read the contents of the LICENSE file
with open("LICENSE", "r", encoding="utf-8") as fh:
    license_text = fh.read()

setuptools.setup(
    name="pacina",
    version="1.0.0",
    description="Power measurement driver for PAC+INA devices",
    long_description=long_description,
    long_description_content_type="text/markdown",
    license=license_text,
    python_requires=">=3.8",
    install_requires=[
        "pyftdi",
    ],
    extras_require={
        "test": ["pytest"],
        "viz": ["pandas", "plotly"],
    },
    packages=setuptools.find_packages(),
    entry_points={
        "console_scripts": [
            "pacina-cli = pacina.cli:main",
        ],
    },
)
