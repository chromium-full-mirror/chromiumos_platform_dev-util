// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use std::{
    io::{stdout, Write},
    sync::{atomic::AtomicBool, Arc, RwLock},
};

use log::warn;

static SHUTDOWN_TRIGGERED: RwLock<Option<Arc<AtomicBool>>> = RwLock::new(None);
static SHUTDOWN_WARNING_PRINTED: RwLock<Option<Arc<AtomicBool>>> = RwLock::new(None);

pub fn setup_signals() {
    // Set the global signal handling, one CTRL-C is graceful, two is immediate.
    let trigger_bool: Arc<AtomicBool> = Arc::new(AtomicBool::new(false));
    let message_bool: Arc<AtomicBool> = Arc::new(AtomicBool::new(false));
    let _ = SHUTDOWN_TRIGGERED
        .write()
        .unwrap()
        .insert(Arc::clone(&trigger_bool));
    let _ = SHUTDOWN_WARNING_PRINTED
        .write()
        .unwrap()
        .insert(Arc::clone(&message_bool));
    signal_hook::flag::register_conditional_shutdown(
        signal_hook::consts::SIGINT,
        1,
        Arc::clone(&trigger_bool),
    )
    .unwrap();
    signal_hook::flag::register(signal_hook::consts::SIGINT, Arc::clone(&trigger_bool)).unwrap();
}

pub fn is_shutting_down() -> bool {
    let shutting_down = SHUTDOWN_TRIGGERED.read().unwrap();
    let message_printed = SHUTDOWN_WARNING_PRINTED.write().unwrap();
    let shutting_down = shutting_down.as_ref().unwrap();
    let message_printed = message_printed.as_ref().unwrap();
    let going_down = shutting_down.load(std::sync::atomic::Ordering::Relaxed);
    if going_down && !message_printed.load(std::sync::atomic::Ordering::Relaxed) {
        message_printed.store(true, std::sync::atomic::Ordering::Relaxed);
        warn!("Shutting down gracefully, long running syncs may seem hung, press ctrl-c again to force");
        stdout().flush().unwrap()
    }
    going_down
}
