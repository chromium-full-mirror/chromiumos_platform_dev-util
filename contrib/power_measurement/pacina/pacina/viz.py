# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Visualization functionality for Pacina measurements."""

import json
import logging
import pathlib
import time
import typing as t
import urllib

from . import cs_types


logger = logging.getLogger(__name__)

# Style for the DataFrame Table HTML generation
_STYLES = [
    {"selector": "tr:hover", "props": [("background-color", "#ffff99")]},
    {
        "selector": "",
        "props": [
            ("border-collapse", "collapse"),
            ("width", "100%"),
            ("font-family", "sans-serif"),
        ],
    },
    {
        "selector": "th",
        "props": [
            ("font-size", "100%"),
            ("face", "Open Sans"),
            ("text-align", "left"),
            ("background-color", "#4a86e8"),
            ("color", "white"),
        ],
    },
    {"selector": "caption", "props": [("caption-side", "bottom")]},
]


def generate_reports(
    start_time: float,
    log_path: pathlib.Path,
    log_prefix: str,
    dut_info: t.Optional[dict],
    data: list,
    avg_power: dict,
    config_rails: t.List[dict],
    power_state: cs_types.PowerState,
    header_record: t.List[str],
) -> None:
    # Importing lazily so that only when function is called pandas is required
    import pandas as pd  # pylint: disable=import-error
    import plotly.express  # pylint: disable=import-error

    summary_path = log_path / (log_prefix + "summary.csv")
    summary_html_path = log_path / (log_prefix + "summary.html")

    df = pd.DataFrame(data, columns=header_record)

    df_avg = pd.DataFrame.from_dict(avg_power, orient="index").reset_index()
    df_avg["power"] = df_avg["acc_power"] / df_avg["count"]
    df_avg.columns = [
        "Rail",
        "Total Accumulated (Raw)",
        "Count",
        "Accumulated Power",
        "Sense Resistor",
        "Average Power (W)",
    ]
    pd.options.display.float_format = "{:,.3f}".format

    df_avg = df_avg.sort_values(by=["Average Power (W)"], ascending=False)
    print(df_avg)
    df_avg.to_csv(summary_path)

    # Default Plots
    power_plots = []
    time_plot = plotly.express.line(
        df,
        x="time_relative",
        y="power",
        color="rail",
        labels={"power": "Power (w)", "time_relative": "Time (seconds)"},
    )
    time_plot.update_layout(
        title="Time Series",
        xaxis_title="Time (Seconds)",
        yaxis_title="Power (W)",
    )
    power_plots.append(time_plot)

    box_plot = plotly.express.box(df, y="power", x="rail")
    box_plot.update_layout(
        title="Measurement Statistics",
        xaxis_title="Rail",
        yaxis_title="Power (W)",
    )
    power_plots.append(box_plot)

    sdf_avg = df_avg.style.hide(axis="index").set_table_styles(_STYLES)

    if len(config_rails) > 0:
        rail_map = pd.DataFrame(config_rails)
        df_avg = pd.merge(df_avg, rail_map, on="Rail")
        df_avg["Average Power (W)"] = df_avg["Average Power (W)"].abs()
        df_avg["voltage (mV)"] = df_avg.Rail.apply(
            lambda x: x.split("_")[0].strip("PP")
        )
        star_plot = plotly.express.sunburst(
            df_avg,
            names="Rail",
            parents="Parent",
            values="Average Power (W)",
            title="Power Sunburst",
            color="voltage (mV)",
        )
        power_plots.append(star_plot)

        root_rail = "PPVAR_SYS"
        root_pwr = None
        if root_rail in df_avg.Rail.values:
            root_pwr = df_avg[df_avg.Rail == root_rail][
                "Average Power (W)"
            ].unique()[0]
        else:
            print(root_rail + " measurement not found.")

        t1_columns = ["Rail", "voltage (mV)", "Average Power (W)"]
        if root_rail in df_avg.Parent.values:
            t1_rails = df_avg[df_avg["Parent"] == root_rail]
            t1_pwr = t1_rails["Average Power (W)"].sum()
            print("Tier1 Summary")
            print(f"{'T1 Rail Total:':<20}{t1_pwr:>20.3f}")
            if root_pwr:
                print(f"{'T1 Root %s' % root_rail:<20}{root_pwr:>20.3f}")
                print(f"{'Root - T1 Total:':<20}{(root_pwr - t1_pwr):>20.3f}")

            print(t1_rails[t1_columns])

    with summary_html_path.open("w") as f:
        f.write("<h1>" + summary_html_path.name + "<h1>")
        if dut_info:
            test_info_path = log_path / (log_prefix + "test-info.json")
            stop_time = time.strftime("%Y-%m-%d %H:%M:%S")
            test_info: dict = {}
            test_info["id"] = log_prefix
            test_info["measurement_phase"] = str(power_state)
            test_info["measurement_start_time"] = time.strftime(
                "%Y%m%d %H%M%S", time.localtime(start_time)
            )
            test_info["measurement_stop_time"] = stop_time
            test_info = test_info | dut_info
            del test_info["configs"]
            del test_info["ftdi_urls"]

            df_test_info = (
                pd.DataFrame(test_info, index=[0]).transpose().reset_index()
            )
            df_test_info.columns = ["Items", "Test Config"]

            s = df_test_info.style.hide(axis="index").set_table_styles(_STYLES)

            f.write(s.render())

            test_info["upload_data"] = True
            with open(test_info_path, "w", encoding="utf-8") as fjson:
                json.dump(test_info, fjson, indent=2, separators=(",", ": "))

        f.write(sdf_avg.to_html())
        for pplts in power_plots:
            f.write(
                pplts.to_html(
                    full_html=False,
                    include_plotlyjs="cdn",
                    default_height="50%",
                )
            )
    report_file_path = urllib.parse.quote(  # type: ignore[attr-defined]
        str(summary_html_path.absolute())
    )
    print(f"Report: file://{report_file_path}")


def show_log_single_data(
    data: list, gpio_data: list, log_single_header: t.List[str]
) -> None:
    # Importing lazily so that only when function is called pandas is required
    import pandas as pd  # pylint: disable=import-error

    df = pd.DataFrame(data, columns=log_single_header)

    pd.options.display.float_format = "{:,.3f}".format
    print(df)

    if gpio_data:
        df_gpio = pd.DataFrame(gpio_data, columns=["GPIO", "State"])
        print("\nGPIO States")
        print(df_gpio)
