// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The repo format is described here:
// https://gerrit.googlesource.com/git-repo/+/master/docs/manifest-format.md

extern crate anyhow;
extern crate backoff;
extern crate chrono;
extern crate clap;
extern crate git2;
extern crate humantime;
extern crate log;
extern crate rayon;
extern crate regex;
extern crate serde;
extern crate serde_json;
extern crate tempfile;
extern crate url;
extern crate xml;

use auth::initialize_gitcookies;
use chrono::{DateTime, Utc};
use clap::{Args, Parser, Subcommand};
use log::{error, info, trace};
use progress::initialize_progresses;
use root::create_root;
use std::env::current_dir;
use std::path::Path;
use std::process;
use std::sync::RwLock;
use url::Url;

use crate::error::{RootError, SnagError};
use crate::manifest::parse_manifest;
use crate::root::{discover_root, load_root};
use crate::signals::setup_signals;
use crate::sync::sync;
use crate::util::{level_filter_from_int, rounded_elapsed_runtime, set_start_time};

mod auth;
mod error;
mod manifest;
mod progress;
mod root;
mod signals;
mod sync;
mod util;

static PROG_START_TIME: RwLock<Option<DateTime<Utc>>> = RwLock::new(None);

const LONG_ABOUT: &str = "
Snag is a utility for manipulating large numbers of git repositories. It
follows the manifest convention of `repo` which it shares many of the
same purposes. It doesn't aim for 100% compatibility for all features
and nuiances of the manifest format or `repo`, but to replicate the common
workflows of working with source.

Snag is fast, safe, and easy to use.";

#[derive(Debug, Parser)]
#[clap(name = "snag", version, long_about=LONG_ABOUT)]
pub struct AppArgs {
    #[clap(flatten)]
    global_opts: GlobalOpts,

    #[clap(subcommand)]
    command: Command,
}

#[derive(Debug, Args)]
struct GlobalOpts {
    /// The thread count to use when possible. Default = N_CPUS2.
    #[clap(short, long, global = true, required = false)]
    jobs: Option<usize>,

    /// The root directory (e.g. /home/engeg/chromiumos).
    #[clap(short, long, global = true, required = false)]
    root: Option<String>,

    /// The logging level (Off=0, Error=1, Warn=2, Info=3, Debug=4, or Trace=5).
    #[clap(short, long, global = true, required = false, default_value_t = 3)]
    verbosity: u32,

    /// Path to the netscape style cookies file.
    #[clap(
        short,
        long,
        global = true,
        required = false,
        default_value = "~/.gitcookies"
    )]
    cookie_path: String,
}

#[derive(Debug, Subcommand)]
enum Command {
    /// Pull the latest source from the remotes.
    Sync {
        /// Specify a manifest to load.
        #[clap(short, long)]
        manifest: Option<String>,

        /// Use force (including discarding changes) so local reflects the remote.
        #[clap(short, long)]
        force: bool,

        /// Only perform the fetch portion, do not checkout contents.
        #[clap(long)]
        fetch_only: bool,

        /// Which groups should we sync, can be specified multiple times.
        #[clap(short, long)]
        group: Option<Vec<String>>,

        /// Do not update the manifests from the tracking ref before syncing.
        #[clap(short, long)]
        stasis: bool,
    },

    /// Initialize a snag root and set an upstream manifest.
    Init {
        /// Clobber the existing contents, clearing all metadata and config.
        #[clap(long)]
        clobber: bool,

        /// Track the following revision upsteam.
        #[clap(short, long, default_value = "refs/heads/main")]
        tracking_ref: String,

        /// Manifest URL to which contains the .xml definitions.
        #[clap(short, long)]
        url: String,

        /// Manifest file to load from the repository.
        #[clap(short, long, default_value = "default.xml")]
        manifest: String,
    },

    /// Get the status of every local repository.
    Status {},
}

/// If we're using multithreaded subcommands, initialize a rayon threadpool.
fn init_thread_pool(num_threads: usize) {
    // Set up the global thread pool.
    rayon::ThreadPoolBuilder::new()
        .num_threads(num_threads)
        .build_global()
        .expect("Could not start threadpool");
}

fn run() -> Result<(), anyhow::Error> {
    // Set the start time.
    set_start_time();

    // Process command line arguments.
    let args = AppArgs::parse();
    let num_threads = args.global_opts.jobs.unwrap_or(num_cpus::get());

    let mut builder = env_logger::Builder::from_default_env();
    builder
        .filter(None, level_filter_from_int(args.global_opts.verbosity))
        .init();

    // Set cwd.
    let cwd = current_dir().expect("Could not get current directory");
    let cwd = cwd.to_str().expect("Current directory is not valid");

    // Catch ctrl-c gracefully.
    setup_signals();

    // Give git callbacks have somewhere to write status.
    initialize_progresses();

    // Prepare auth.
    match initialize_gitcookies(Some(args.global_opts.cookie_path)) {
        Ok(_) => {}
        Err(e) => return Err(e.into()),
    }

    // Set up the root, discover if possible, otherwise create.
    let root_dir = args.global_opts.root.unwrap_or(String::from(cwd));
    let root_dir = shellexpand::tilde(&root_dir).into_owned();
    let root_dir = Path::new(&root_dir);

    match args.command {
        // The sync command.
        Command::Sync {
            manifest,
            force,
            fetch_only,
            group,
            stasis,
        } => {
            init_thread_pool(num_threads);
            let snag_root = match discover_root(root_dir.to_str().unwrap()) {
                Some(r) => match load_root(&r) {
                    Ok(r) => r,
                    Err(e) => return Err(e),
                },
                None => {
                    error!("No snag root was found");
                    return Err(RootError::NotFound.into());
                }
            };

            // Sync the manifest.
            if !stasis {
                info!("Syncing upstream manifest");
                match snag_root.sync_root() {
                    Ok(r) => {
                        trace!("{:#?}", r);
                    }
                    Err(e) => return Err(e),
                }
            }

            // Parse the manifest.
            let manifest = parse_manifest(
                &snag_root
                    .manifest_path()
                    .join(manifest.unwrap_or(snag_root.manifest.clone())),
            );

            trace!("{:#?}", manifest);
            info!("Syncing projects into {:?}", snag_root.path);
            let _sync_results = sync(manifest, snag_root.path.into(), force, fetch_only, group)
                .expect("No sync results returned");
            info!("Total Sync Duration: {}", rounded_elapsed_runtime());
            Ok(())
        }

        // The status command.
        Command::Status {} => {
            let e = SnagError::NotImplemented;
            error!("{:#?}", e);
            Err(e.into())
        }

        // The init command.
        Command::Init {
            clobber,
            tracking_ref,
            url,
            manifest,
        } => match Url::parse(url.as_str()) {
            Ok(manifest_url) => {
                let root_dir = root_dir.to_str().unwrap();
                match create_root(root_dir, &manifest_url, tracking_ref, clobber, manifest) {
                    Ok(snag_root) => {
                        info!(
                            "Created snag root at {}, syncing the specified manifest...",
                            root_dir
                        );
                        match snag_root.sync_root() {
                            Ok(sr) => match sr.err {
                                Some(e) => Err(e.into()),
                                None => Ok(()),
                            },
                            Err(e) => Err(e),
                        }
                    }
                    Err(e) => Err(e),
                }
            }
            Err(e) => {
                error!("The url provided could not be parsed");
                Err(e.into())
            }
        },
    }
}

fn main() {
    let code = match run() {
        Ok(_) => 0,
        Err(e) => {
            error!("{:#?}", e);
            1
        }
    };
    process::exit(code)
}
