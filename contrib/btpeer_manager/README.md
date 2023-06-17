# `btpeer_manager`

A CLI application that makes managing btpeers in our lab easier by automating
common operations including:

* Building chameleond and preparing the bundle for btpeer deployment
* Chameleond bundle release management, handling the config logic and GCS access

## Installation

Use the `./install.sh` bash script to build and install btpeer_manager so that
it may be run as a regular command, `btpeer_manager`. The built copy will reside
at `./bin/btpeer_manager` and a system link to this executable will be placed
in your desired install directory. The default install dir is `~/lib/depot_tools`
to prevent the need of adding another dir to your bash PATH, but it can be
adjusted if desired (see below).

Note: The installation script is meant to be run on your main system, not in a
chroot.

Note: Syncing an updated version of this source repository will not
automatically rebuild an updated version of `btpeer_manager`. To
update your local build, simply re-run `./install.sh` (or `./build.sh`) to
rebuild it.

```text
$ bash ./install.sh help
Usage: install.sh [options]

Options:
 --dir|-d <path>    Path to directory where btpeer_manager is to be
                    installed, which should be in your $PATH
                    (default = '~/lib/depot_tools').
```
