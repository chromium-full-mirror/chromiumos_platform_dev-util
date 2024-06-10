# Pacina

Power measurement driver for PAC+INA devices.

For extensive analysis of the internal workings and performance of the script
and PAC device, see go/pac-power-measurement.

For general description of the Cros-Power-Debugger (CPD), see go/cpd-care.

## Installation

### Setting udev rules

Script assumes that FTDI USB-to-I2C device(s) are used to communicate to and
from the current sensor ICs. As such, appropriate udev permissions need to be
set. Run:

    $ sudo cp 11-ftdi.rules /etc/udev/rules.d/11-ftdi.rules

### Building python package

To install the package and its dependencies, run:

    $ pip install .

If you want to generate viz and html reports:

    $ pip install ".[viz]"

To also run tests:

    $ pip install ".[test]"

### Logging power

To see the usage, run:

    $ ./run.sh --help

Example of cpd.py config file can be found in pacina/cpd.example.py

### Outputs
Following outputs are generated when taking a single measurement (`-s, --single`):
* singleLog.csv containing instantaneous voltage, current and power readings for the rails specified in the config(s).

Following outputs are generated when taking continuous measurements:
* timeLog.csv - raw power reads for the rails specified in the config(s).
* summary.csv - summary of the average power for the rails specified in the config(s).
* summary.html - containing rail time series plots and summary tables.

If `dut_info` is provided, following additional file is generated:
* testinfo.json - containing DUT and test related information.

### Benchmarks

To generate the benchmarks, run:

    $ ./benchmark_pacina.sh

For more details about it, please see the docstring inside the script.
