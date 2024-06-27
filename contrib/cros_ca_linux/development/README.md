# Development Guidelines
## How to add new dependencies into Linux venv

Sometimes we need to add a new dependency to Linux venv. This includes adding new dependencies to CrOS venv because currently we push the Linux venv to CrOS.

Follow these procedure to add new python dependencies.

1. Create and go into the virtual env under cros_ca_linux direcorty.
```
> source script/setup_venv.sh .
```

2. Install the new dependencies under venv.
```
> pip install <package>
```

3. Freeze all the dependencies into development/requirements.in file
```
> pip freeze > development/requirements.in
```

4. Install toolings for generating dependency hashes.
```
> pip install --require-hashes --no-deps -r development/base-tooling-requirements.txt
```

5. Generate requirements.txt with hashes.
```
> pip-compile --generate-hashes --output-file ./requirements.txt development/requirements.in
```

A new requirements.txt file will be generated.

6. Commit the changes.

Git commit the changes in development/requirements.in and requirements.txt.

## How to add new dependencies into Windows venv

Prerequisites:

- Windows system
- Installed Python 3.8

Run the following command in the powershell console.

1. Install poetry if not yet installed.
```
> python -m pip install --user poetry
```

2. Create virtual env by poetry.

Go to the `win` directory under the source repository and use Poetry to create the virtual environment.
```
> cd win
> python -m poetry install
```

3. Install the new dependencies.
```
> python -m poetry add <package>
```

4. Commit the changes of the files `win/poetry.lock` and `win/pyproject.toml`.
