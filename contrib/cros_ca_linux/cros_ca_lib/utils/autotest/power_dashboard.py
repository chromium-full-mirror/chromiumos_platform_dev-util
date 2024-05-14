# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Power dashboard related functions."""

import json
import os
import sys


_HTML_CHART_STR = """
<!DOCTYPE html>
<html>
<head>
<script type="text/javascript" src="https://www.gstatic.com/charts/loader.js">
</script>
<script type="text/javascript">
    google.charts.load('current', {{'packages':['corechart', 'table']}});
    google.charts.setOnLoadCallback(drawChart);
    function drawChart() {{
        var dataArray = [
{data}
        ];
        var data = google.visualization.arrayToDataTable(dataArray);
        var numDataCols = data.getNumberOfColumns() - 1;
        var unit = '{unit}';
        var options = {{
            width: 1600,
            height: 1200,
            lineWidth: 1,
            legend: {{ position: 'top', maxLines: 3 }},
            vAxis: {{ viewWindow: {{min: 0}}, title: '{type} ({unit})' }},
            hAxis: {{ viewWindow: {{min: 0}}, title: 'time (second)' }},
        }};
        var element = document.getElementById('{type}');
        var chart;
        if (unit == 'percent') {{
            options['isStacked'] = true;
            if (numDataCols == 2) {{
                options['colors'] = ['#d32f2f', '#43a047']
            }} else if (numDataCols <= 4) {{
                options['colors'] = ['#d32f2f', '#f4c7c3', '#cddc39','#43a047'];
            }} else if (numDataCols <= 9) {{
                options['colors'] = ['#d32f2f', '#e57373', '#f4c7c3', '#ffccbc',
                        '#f0f4c3', '#c8e6c9', '#cddc39', '#81c784', '#43a047'];
            }}
            chart = new google.visualization.SteppedAreaChart(element);
        }} else if (data.getNumberOfRows() == 2 && unit == 'point') {{
            var newArray = [['key', 'value']];
            for (var i = 1; i < dataArray[0].length; i++) {{
                newArray.push([dataArray[0][i], dataArray[1][i]]);
            }}
            data = google.visualization.arrayToDataTable(newArray);
            delete options.width;
            delete options.height;
            chart = new google.visualization.Table(element);
        }} else {{
            chart = new google.visualization.LineChart(element);
        }}
        chart.draw(data, options);
    }}
</script>
</head>
<body>
<div id="{type}"></div>
</body>
</html>
"""


def save_html(
    metricfile,
    time_metric,
    resultsdir,
    one_value_metrics=None,
    filename="power_log.html",
):
    """Convert metric result file to chart in HTML page.

    Note that this results in multiple HTML objects in one file but Chrome
    can render all of it in one page.

    Args:
        metricfile: results-chart.json file path
        time_metric: the metric name for time recording, such as "Browsing.t".
                    All other metrics that use this timeline will be saved into
                    the HTML chart.
        resultsdir: directory to save HTML page
        one_value_metrics: list of one-value metrics that to be added to the
                    timeline. One value metric will show a horizontal line
                    in the chart.
        filename: filename of the HTML page
    """
    rail_type = {}
    powerlog_dict = {
        "power": {
            "sample_count": 0,
            "sample_duration": 0,
            "data": {},
            "unit": {},
        }
    }
    time_data = []
    # time_unit = "s"
    one_value_rail = "One-value metrics"
    with open(metricfile, "r", encoding="utf-8") as f:
        data = json.load(f)
        time_metric_found = False
        for m in data:
            if "summary" not in data[m]:
                # Currently only check the "summary".
                continue
            summary = data[m]["summary"]
            if m == time_metric:
                time_metric_found = True
                time_data = summary["values"]
                # time_unit = summary["units"]
                continue
            if "interval" in summary and summary["interval"] == time_metric:
                # Find the metric using the given timeline.
                powerlog_dict["power"]["data"][m] = summary["values"]
                powerlog_dict["power"]["unit"][m] = summary["units"]
                # Multiple metrics have the same type (memory, CPU, etc).
                # But for now, we assign each metric to its own type.
                rail_type[m] = [m]
        if not time_metric_found:
            raise Exception(f"time metric {time_metric} is not found")
        sample_count = len(time_data)
        for m, samples in powerlog_dict["power"]["data"].items():
            if len(samples) != sample_count:
                raise Exception(
                    f"{m} has different sample count with {time_metric}"
                )
        powerlog_dict["power"]["sample_count"] = sample_count
        powerlog_dict["power"]["sample_duration"] = (
            time_data[-1] - time_data[0] / sample_count
        )
        # Handle one-value metrics:
        if not one_value_metrics:
            one_value_metrics = []
        # Data for HTML table
        one_value_keys = []
        for m in one_value_metrics:
            if m not in data:
                continue
            if "summary" not in data[m]:
                # Currently only check the "summary".
                continue
            summary = data[m]["summary"]
            # Generated the data needed for the line chart in HTML
            powerlog_dict["power"]["data"][m] = [
                summary["value"]
            ] * sample_count  # Repeat the one-value along the timeline.
            powerlog_dict["power"]["unit"][m] = summary["units"]
            # Generated the data needed for the table in HTML.
            one_value_keys.append(m)
        if len(one_value_keys) > 0:
            rail_type[one_value_rail] = one_value_keys
    # TODO: if time_unit is not second, convert the time_data to second.
    html_str = ""
    row_indent = " " * 12
    for k, v in rail_type.items():
        data_str_list = []
        # Generate rail name data string.
        header = ["time"] + v
        header_str = row_indent + "['" + "', '".join(header) + "']"
        data_str_list.append(header_str)
        sample_range = range(powerlog_dict["power"]["sample_count"])
        if k == one_value_rail:
            # Only generate two rows of data.
            sample_range = [0, powerlog_dict["power"]["sample_count"] - 1]
        # Generate measurements data string.
        for i in sample_range:
            row = [str(time_data[i])]  # Use the timeline value for the time.
            for r in v:
                row.append(str(powerlog_dict["power"]["data"][r][i]))
            row_str = row_indent + "[" + ", ".join(row) + "]"
            data_str_list.append(row_str)
        data_str = ",\n".join(data_str_list)
        unit = powerlog_dict["power"]["unit"][v[0]]
        if k == one_value_rail:
            unit = "point"
        html_str += _HTML_CHART_STR.format(data=data_str, unit=unit, type=k)
    if not os.path.exists(resultsdir):
        raise Exception(f"resultsdir {resultsdir} does not exist.")
    filename = os.path.join(resultsdir, filename)
    with open(filename, "a", encoding="utf-8") as f:
        f.write(html_str)


def main(argv):
    metric_file = argv[0]
    time_metric = argv[1]
    resultsdir = argv[2]
    save_html(metric_file, time_metric, resultsdir)


if __name__ == "__main__":
    main(sys.argv[1:])
