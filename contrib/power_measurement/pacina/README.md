## Prerequisite

Script assumes that FTDI USB-to-I2C device(s) are used to communicate to and
from the current sensor ICs. As such, appropriate udev permissions need to be
set.

### Setting udev rules

Add the following in `/etc/udev/rules.d/11-ftdi.rules`

```
SUBSYSTEM=="usb", ATTR{idVendor}=="0403", GROUP="plugdev", MODE="0660"`
SUBSYSTEM=="usb_device", ATTR{idVendor}=="0403", GROUP="plugdev", MODE="0660"
```

### Python Setup

Script can be run both inside and outside of chroot and requires the following packages:

```
pandas
plotly
pyftdi
```

If running in chroot, run the following commands to install the required
packages:

```
wget https://bootstrap.pypa.io/pip/3.6/get-pip.py
sudo python ./get-pip.py
sudo pip3 install pandas plotly pyftdi
```

It is recommended to run the script outside of chroot, as the setup described in
this documennt requires to run `run.sh` in `sudo`.

### Logging power

To see the usage, run:

    ./run.sh --help

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

    ./benchmark_pacina.sh

For more details about it, please see the docstring inside the script.
