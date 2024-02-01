// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
extern crate console;
extern crate indicatif;

use std::{
    collections::HashMap,
    path::Path,
    sync::{Arc, Mutex, RwLock},
    thread,
    time::Duration,
};

use self::console::Term;
use self::indicatif::{MultiProgress, ProgressBar, ProgressDrawTarget, ProgressStyle};
use git2::{PackBuilderStage, Progress};
use log::{debug, error, LevelFilter};

use crate::{
    signals::is_shutting_down,
    sync::{SyncOutcome, SyncResult},
    util::rounded_elapsed_runtime,
};

// Style our progress bars.
const BAR_STYLE_TEMPLATE: &str = "[{elapsed_precise}] {bar:60.cyan/blue} {pos:>7}/{len:7} {msg}";
const BAR_PROGRESS_CHARS: &str = "##-";

// Keep a hash map of the latest reported statuses per project_path.
static PROGRESSES: RwLock<Option<HashMap<String, GitProgressHolder>>> = RwLock::new(None);

#[derive(Default)]
struct GitProgressHolder {
    path: String,
    index_total: usize,
    index_current: usize,
    index_bar: Option<ProgressBar>,

    pack_total: usize,
    pack_current: usize,
    pack_bar: Option<ProgressBar>,

    checkout_total: usize,
    checkout_current: usize,
    checkout_bar: Option<ProgressBar>,
}

impl GitProgressHolder {
    fn update_progress_bars(&mut self, main_bar: &MultiProgress) {
        {
            // Index bar.
            if self.index_bar.is_none() {
                let pb = main_bar.add(ProgressBar::new(self.index_total as u64));
                pb.set_style(
                    ProgressStyle::with_template(BAR_STYLE_TEMPLATE)
                        .unwrap()
                        .progress_chars(BAR_PROGRESS_CHARS),
                );
                pb.set_message(format!("indexing -- {}", self.path));
                self.index_bar = Some(pb);
            }
            let ib = self.index_bar.as_ref().unwrap();
            ib.set_position(self.index_current as u64);
            if self.index_current == self.index_total {
                ib.finish_and_clear();
            }
        }
        {
            // Pack bar.
            if self.pack_bar.is_none() {
                let pb = main_bar.add(ProgressBar::new(self.pack_total as u64));
                pb.set_style(
                    ProgressStyle::with_template(BAR_STYLE_TEMPLATE)
                        .unwrap()
                        .progress_chars(BAR_PROGRESS_CHARS),
                );
                pb.set_message(format!("packing -- {}", self.path));
                self.pack_bar = Some(pb);
            }
            let ib = self.pack_bar.as_ref().unwrap();
            ib.set_position(self.pack_current as u64);
            if self.pack_total == self.pack_current {
                ib.finish_and_clear();
            }
        }
        {
            // Checkout bar.
            if self.checkout_bar.is_none() {
                let pb = main_bar.add(ProgressBar::new(self.checkout_total as u64));
                pb.set_style(
                    ProgressStyle::with_template(BAR_STYLE_TEMPLATE)
                        .unwrap()
                        .progress_chars(BAR_PROGRESS_CHARS),
                );
                pb.set_message(format!("checking out -- {}", self.path));
                self.checkout_bar = Some(pb);
            }
            let ib = self.checkout_bar.as_ref().unwrap();
            ib.set_position(self.checkout_current as u64);
            if self.checkout_current == self.checkout_total {
                ib.finish_and_clear();
            }
        }
    }
}

pub fn initialize_progresses() {
    let mut prog = PROGRESSES.write().unwrap();
    if prog.is_some() {
        error!("You should not call initialize_progresses twice, will not reload");
    } else {
        *prog = Some(HashMap::new());
    }
}

pub fn add_sync(ongoing_syncs: &Arc<Mutex<Vec<std::string::String>>>, path: &str) {
    // Record the name of the job, in a new scope to drop the lock.
    ongoing_syncs.lock().unwrap().push(path.to_string());
}

pub fn rm_sync(ongoing_syncs: &Arc<Mutex<Vec<std::string::String>>>, path: &str) {
    // Filter out this path from the ongoing syncs.
    ongoing_syncs.lock().unwrap().retain(|x| *x != path);

    // Finalize the associated progress bars.
    let mut prog_map = PROGRESSES.write().unwrap();
    let prog_map = prog_map.as_mut().unwrap();
    if let Some(pb) = prog_map.get(path) {
        match &pb.checkout_bar {
            Some(b) => b.finish_and_clear(),
            None => {}
        }
        match &pb.index_bar {
            Some(b) => b.finish_and_clear(),
            None => {}
        }
        match &pb.pack_bar {
            Some(b) => b.finish_and_clear(),
            None => {}
        }
    }
}

pub fn simple_xfer_progress(project_path: &str, progress: Progress) -> bool {
    let mut prog_map = PROGRESSES.write().unwrap();
    let prog_map = prog_map.as_mut().unwrap();
    let ph = match prog_map.get_mut(project_path) {
        Some(p) => p,
        None => {
            prog_map.insert(
                project_path.to_string(),
                GitProgressHolder {
                    path: project_path.to_string(),
                    ..GitProgressHolder::default()
                },
            );
            prog_map.get_mut(project_path).unwrap()
        }
    };
    ph.index_current = progress.indexed_objects();
    ph.index_total = progress.total_objects();

    true
}

pub fn simple_pack_progress(
    project_path: &str,
    _pbs: PackBuilderStage,
    current: usize,
    total: usize,
) {
    let mut prog_map = PROGRESSES.write().unwrap();
    let prog_map = prog_map.as_mut().unwrap();
    let ph = match prog_map.get_mut(project_path) {
        Some(p) => p,
        None => {
            prog_map.insert(
                project_path.to_string(),
                GitProgressHolder {
                    path: project_path.to_string(),
                    ..GitProgressHolder::default()
                },
            );
            prog_map.get_mut(project_path).unwrap()
        }
    };
    ph.pack_current = current;
    ph.pack_total = total;
}

pub fn simple_checkout_progress(
    project_path: &str,
    _path: Option<&Path>,
    current: usize,
    total: usize,
) {
    let mut prog_map = PROGRESSES.write().unwrap();
    let prog_map = prog_map.as_mut().unwrap();
    let ph = match prog_map.get_mut(project_path) {
        Some(p) => p,
        None => {
            prog_map.insert(
                project_path.to_string(),
                GitProgressHolder {
                    path: project_path.to_string(),
                    ..GitProgressHolder::default()
                },
            );
            prog_map.get_mut(project_path).unwrap()
        }
    };
    ph.checkout_current = current;
    ph.checkout_total = total;
}

pub fn overall_status_thread(
    sync_results: Arc<Mutex<Vec<SyncResult>>>,
    ongoing_syncs: Arc<Mutex<Vec<std::string::String>>>,
    manifest_len: usize,
) {
    thread::spawn(move || {
        let bars = MultiProgress::new();
        bars.set_draw_target(ProgressDrawTarget::term(Term::buffered_stdout(), 1));
        if log::max_level() > LevelFilter::Info {
            bars.set_draw_target(ProgressDrawTarget::hidden())
        }

        let overall_bar = bars.add(ProgressBar::new(manifest_len as u64));
        overall_bar.set_style(
            ProgressStyle::with_template(BAR_STYLE_TEMPLATE)
                .unwrap()
                .progress_chars(BAR_PROGRESS_CHARS),
        );

        let mut loop_i: u64 = 0;
        loop {
            // Keep the shutting down status up to date by polling it.
            if is_shutting_down() {
                break;
            }

            // Get the lock and drop it as soon as possible.
            let ongoing_syncs_n;
            {
                ongoing_syncs_n = ongoing_syncs.lock().unwrap().len();
            }

            let results_len: usize = {
                let results = sync_results.lock().unwrap();
                let results_len = results.len();

                let n_failures = results
                    .iter()
                    .filter(|x| x.outcome != SyncOutcome::Success)
                    .collect::<Vec<_>>()
                    .len();

                let rer = rounded_elapsed_runtime();

                // Handle the overall progress bar.
                let bar_msg = format!(
                    "Totals: {n_failures} failures, {ongoing_syncs_n} running, {rer} elapsed"
                );
                overall_bar.set_position(results_len as u64);
                overall_bar.set_message(bar_msg.clone());

                // Only print every 10 seconds (10 x 1000ms).
                if loop_i % 10 == 0 {
                    debug!("Completed {results_len} of {manifest_len} {bar_msg}");
                }

                results.len()
            };

            // Add the in progress checkout sub bars.
            {
                let mut unlocked_progresses = PROGRESSES.write().unwrap();
                for s in ongoing_syncs.lock().unwrap().iter() {
                    let prog_map = unlocked_progresses.as_mut().unwrap();
                    if let Some(s) = prog_map.get_mut(s) {
                        s.update_progress_bars(&bars);
                    }
                }
            }

            thread::sleep(Duration::from_millis(1000));

            if results_len >= manifest_len {
                break;
            }
            loop_i += 1;
        }
    });
}
