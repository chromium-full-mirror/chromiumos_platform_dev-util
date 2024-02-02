// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use chrono::{DateTime, DurationRound, Utc};
use log::LevelFilter;
use regex::RegexBuilder;

use crate::PROG_START_TIME;

pub fn level_filter_from_int(level: u32) -> LevelFilter {
    match level {
        0 => LevelFilter::Off,
        1 => LevelFilter::Error,
        2 => LevelFilter::Warn,
        3 => LevelFilter::Info,
        4 => LevelFilter::Debug,
        5 => LevelFilter::Trace,
        6_u32..=u32::MAX => {
            panic!("invalid logging level")
        }
    }
}

pub fn str_is_sha1(s: &str) -> bool {
    let sha1_regex = RegexBuilder::new(r"^[0-9a-f]{40}$").build().unwrap();
    sha1_regex.is_match(s)
}

pub fn set_start_time() -> DateTime<Utc> {
    let s_time = Utc::now();
    let mut _p_s_time = PROG_START_TIME
        .write()
        .expect("could not set program start time");
    *_p_s_time = Some(s_time);
    s_time
}

pub fn op_duration_ms(s_time: DateTime<Utc>) -> u64 {
    (Utc::now() - s_time).num_milliseconds() as u64
}

pub fn rounded_elapsed_runtime() -> String {
    let s_time = PROG_START_TIME
        .read()
        .expect("cannot read start time")
        .unwrap();
    let s_time = s_time.duration_round(chrono::Duration::seconds(1)).unwrap();

    let now = Utc::now()
        .duration_round(chrono::Duration::seconds(1))
        .unwrap();

    humantime::format_duration(
        (now - s_time)
            .to_std()
            .expect("failed to convert timestamps"),
    )
    .to_string()
}
