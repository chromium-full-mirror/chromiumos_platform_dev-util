# Copyright 2024 The Chromium Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Find the outliers using IQR formula."""

import numpy as np


def calculate_iqr_outliers(use_case_results: list):
    """
    Calculate Inter quartile Range (IQR) and identify outliers in a dataset.

    This function takes a list of numerical values and calculates the first
    and third quartiles to find the IQR and the outliers.

    Args:
        use_case_results (list): A list of numerical values for which to
            calculate IQR and identify outliers.

    Returns:
        dict: A dictionary containing the following key-value pairs:
            - "Q1": The first quartile.
            - "Q3": The third quartile.
            - "IQR": The Inter quartile Range.
            - "Lower Bound": The lower bound for identifying outliers.
            - "Upper Bound": The upper bound for identifying outliers.
    """

    data_array = np.array(use_case_results)
    sorted_data = np.sort(data_array)
    q1_index = int(0.25 * (len(sorted_data)))
    q3_index = int(0.75 * (len(sorted_data)))

    if len(sorted_data) % 2 == 0:
        q1 = (sorted_data[q1_index] + sorted_data[q1_index - 1]) / 2
        q3 = (sorted_data[q3_index] + sorted_data[q3_index - 1]) / 2
    else:
        q1 = sorted_data[q1_index]
        q3 = sorted_data[q3_index]

    # Calculate the IQR value
    iqr = q3 - q1

    # Calculate lower and upper bounds for outliers
    lower_bound = q1 - (1.5 * iqr)
    upper_bound = q3 + 1.5 * iqr
    outliers = (data_array < lower_bound) | (data_array > upper_bound)

    return {
        "Q1": q1,
        "Q3": q3,
        "IQR": iqr,
        "Lower Bound": lower_bound,
        "Upper Bound": upper_bound,
        "Outliers": data_array[outliers].tolist(),
    }


if __name__ == "__main__":
    # Example
    use_case_sample_result = [11, 31, 21, 19, 8, 54, 35, 26, 23, 13, 29, 17]
    result = calculate_iqr_outliers(use_case_sample_result)
