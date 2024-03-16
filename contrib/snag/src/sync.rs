// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use std::{
    io::{stdout, Write},
    path::{Path, PathBuf},
    sync::{Arc, Mutex},
};

use anyhow::Error;
use backoff::{retry_notify, ExponentialBackoff};
use chrono::Utc;
use git2::{
    build::CheckoutBuilder, ErrorClass, ErrorCode, FetchOptions, RemoteCallbacks, Repository,
    RepositoryInitOptions,
};
use log::{debug, error, trace, warn};

use humantime::parse_duration;

use rayon::prelude::{IntoParallelIterator, ParallelIterator};
use url::Url;

use crate::{
    auth::auth_callback,
    error::SyncError,
    manifest::{Manifest, Project},
    progress::{
        add_sync, overall_status_thread, rm_sync, simple_checkout_progress, simple_pack_progress,
        simple_xfer_progress,
    },
    signals::is_shutting_down,
    util::op_duration_ms,
    util::str_is_sha1,
};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum SyncOutcome {
    #[default]
    Unknown = 0,
    Success = 1,
    FetchFailed = 2,
    SetRemoteFailed = 3,
    CheckoutFailed = 4,
    InitializeFailed = 5,
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct SyncResult {
    pub outcome: SyncOutcome,

    pub err: Option<SyncError>,

    pub duration_ms: u64,

    pub name: String,

    pub path: String,

    pub retries: u64,
}

fn create_refspec(remote_name: &str, revision: &str) -> String {
    match str_is_sha1(revision) {
        true => revision.to_string(),
        false => {
            "+".to_owned()
                + revision
                + ":refs/remotes/"
                + remote_name
                + "/"
                + revision
                    .strip_prefix("refs/heads/")
                    .expect("'refs/heads' not prefixing non-sha revision")
        }
    }
}

fn open_or_init_repo(path: &PathBuf) -> Result<Repository, SyncError> {
    // See if the repository already exists (by trying to open it).
    match git2::Repository::open(path.clone()) {
        Ok(r) => Ok(r),
        Err(e) => {
            trace!("Tried to open {:#?}, and got error: {:#?}", path, e);
            let repo_init_options = &mut RepositoryInitOptions::new();
            match git2::Repository::init_opts(path.clone(), repo_init_options) {
                Ok(r) => Ok(r),
                Err(e) => Err(SyncError::OpenOrInit(format!("{:#?}", e))),
            }
        }
    }
}

fn set_remote(
    repo: &Repository,
    remote_url: &str,
    remote_name: &str,
    repo_name: &str,
    refspec: &str,
) -> Result<Url, SyncError> {
    let url = Url::parse(remote_url);
    let url = match url {
        Ok(x) => x,
        Err(e) => return Err(SyncError::SetRemote(format!("{:#?}", e))),
    };

    let full_url = url.join(repo_name);
    let full_url = match full_url {
        Ok(x) => x,
        Err(e) => return Err(SyncError::SetRemote(format!("{:#?}", e))),
    };

    // Delete, and rewrite the remote and refspec.
    match repo.find_remote(remote_name) {
        Ok(_) => match repo.remote_delete(remote_name) {
            Ok(_) => {}
            Err(e) => return Err(SyncError::SetRemote(format!("{:#?}", e))),
        },
        Err(e) => {
            trace!("No existing remote found for {}:\n{:#?}", remote_name, e);
        }
    }

    let fetch_result = repo.remote_with_fetch(remote_name, full_url.as_str(), refspec);
    match fetch_result {
        Ok(_) => {}
        Err(e) => return Err(SyncError::SetRemote(format!("{:#?}", e))),
    };

    Ok(full_url)
}

fn fetch(repo: &Repository, p: &Project, remote_name: &str) -> Result<(), SyncError> {
    let fetch_options = &mut FetchOptions::new();
    fetch_options.prune(git2::FetchPrune::Off);
    fetch_options.download_tags(git2::AutotagOption::Auto);

    let mut cbs = RemoteCallbacks::new();
    cbs.credentials(auth_callback);
    cbs.transfer_progress(|progress| simple_xfer_progress(&p.path, progress));
    cbs.pack_progress(|pbs, current, total| simple_pack_progress(&p.path, pbs, current, total));
    fetch_options.remote_callbacks(cbs);

    // Do the fetch wrapped in exponential backoff.
    let fetch_fn = || -> Result<(), backoff::Error<git2::Error>> {
        let fetch_refspec: &[&str] = &[];
        match repo
            .find_remote(remote_name)
            .expect("remote not found")
            .fetch(fetch_refspec, Some(fetch_options), None)
        {
            Ok(_) => Ok(()),
            Err(e) => match e.code() {
                ErrorCode::Auth | ErrorCode::Certificate => Err(backoff::Error::permanent(e)),
                _ => Err(backoff::Error::transient(e)),
            },
        }
    };

    let notify_fn = |err: git2::Error, _: std::time::Duration| {
        warn!("Retrying transient error: {:?}", err.to_string(),);
    };

    match retry_notify(ExponentialBackoff::default(), fetch_fn, notify_fn) {
        Ok(_) => {}
        Err(e) => return Err(SyncError::Fetch(format!("{:#?}", e))),
    }
    Ok(())
}

fn checkout(
    repo: Repository,
    p: &Project,
    remote_name: &str,
    target_dir: &Path,
    force: bool,
) -> Result<(), SyncError> {
    let full_checkout_ref = p.revision.clone().unwrap();
    let stripped_checkout_ref = full_checkout_ref
        .strip_prefix("refs/heads/")
        .unwrap_or(&full_checkout_ref)
        .to_string();
    let remote_with_ref = remote_name.to_owned() + "/" + &stripped_checkout_ref;
    let is_sha1 = str_is_sha1(&stripped_checkout_ref);

    // Rev parse, deciding if we want the sha alone or with the remote.
    let rev_parse = if is_sha1 {
        stripped_checkout_ref.clone()
    } else {
        remote_with_ref.clone()
    };
    let treeish_obj = match repo.revparse_single(&rev_parse) {
        Ok(x) => x,
        Err(e) => return Err(SyncError::Checkout(format!("Revparse: {:#?}", e))),
    };

    if !is_sha1 {
        // Maybe the branch exists, if not, create it.
        let mut branch = match repo.find_branch(&stripped_checkout_ref, git2::BranchType::Local) {
            Ok(b) => b,
            Err(e) => {
                if e.code() == ErrorCode::NotFound && e.class() == ErrorClass::Reference {
                    match repo.branch(
                        &stripped_checkout_ref,
                        treeish_obj.as_commit().unwrap(),
                        force,
                    ) {
                        Ok(b) => b,
                        Err(e) => return Err(SyncError::Checkout(format!("Branch: {:#?}", e))),
                    }
                } else {
                    return Err(SyncError::Checkout(format!(
                        "Unknown branch error: {:#?}",
                        e
                    )));
                }
            }
        };

        match repo.set_head(&full_checkout_ref) {
            Ok(_) => {}
            Err(e) => return Err(SyncError::Checkout(format!("Set head: {:#?}", e))),
        };

        match branch.set_upstream(Some(&(remote_with_ref.clone()))) {
            Ok(_) => {}
            Err(e) => return Err(SyncError::Checkout(format!("Set upstream: {:#?}", e))),
        };
    } else {
        // Pinned to SHA.
        match repo.set_head_detached(treeish_obj.id()) {
            Ok(_) => {}
            Err(e) => return Err(SyncError::Checkout(format!("Set head detatched: {:#?}", e))),
        };
    }

    let mut cb = CheckoutBuilder::new();
    cb.progress(|path, current, total| simple_checkout_progress(&p.path, path, current, total));
    cb.target_dir(target_dir);
    if force {
        cb.force();
    }

    match repo.checkout_head(Some(&mut cb)) {
        Ok(_) => Ok(()),
        Err(e) => {
            if e.code() == ErrorCode::Conflict && e.class() == ErrorClass::Checkout {
                Err(SyncError::Checkout(format!("Checkout head: {:#?}", e)))
            } else {
                Ok(())
            }
        }
    }
}

fn push_result(sync_results: &Arc<Mutex<Vec<SyncResult>>>, sync_result: &SyncResult) {
    // Report the result on stdout (if unsuccessful).
    if sync_result.outcome != SyncOutcome::Success {
        debug!("Failed sync:\n{:#?}", sync_result);
    } else {
        let readable_time = parse_duration(format!("{} msec", sync_result.duration_ms).as_str())
            .unwrap_or(std::time::Duration::new(0, 0));
        debug!("Finished {} in {:?}", sync_result.path, readable_time);
    }
    stdout().flush().unwrap();

    sync_results
        .lock()
        .expect("failed to get sync_results mutex")
        .push(sync_result.clone());
}

/// Sync one repository. Sync failures are returned in Ok(SyncResult). Errors that
/// are not expected are returned in an Err(Anyhow).
pub fn sync_one(
    p: Project,
    root_path: PathBuf,
    fetch_url: &Url,
    fetch_only: bool,
    remote_name: &str,
    force: bool,
) -> Result<SyncResult, Error> {
    let s_time = Utc::now();

    let sync_result = SyncResult {
        path: p.path.clone(),
        name: p.name.clone(),
        ..Default::default()
    };

    // Initialize the repo.
    let repo_path = root_path.join(Path::new(p.path.as_str()));
    let repo = match open_or_init_repo(&repo_path) {
        Ok(r) => r,
        Err(e) => {
            return Ok(SyncResult {
                outcome: SyncOutcome::InitializeFailed,
                err: Some(e),
                duration_ms: op_duration_ms(s_time),
                ..sync_result
            })
        }
    };

    let refspec = create_refspec(remote_name, &p.revision.clone().unwrap());

    match set_remote(&repo, fetch_url.as_str(), remote_name, &p.name, &refspec) {
        Ok(_) => match fetch(&repo, &p, remote_name) {
            Ok(_) => match fetch_only {
                true => Ok(SyncResult {
                    outcome: SyncOutcome::Success,
                    duration_ms: op_duration_ms(s_time),
                    ..sync_result
                }),
                false => match checkout(repo, &p, remote_name, &repo_path, force) {
                    Ok(_) => Ok(SyncResult {
                        outcome: SyncOutcome::Success,
                        duration_ms: op_duration_ms(s_time),
                        ..sync_result
                    }),
                    Err(e) => Ok(SyncResult {
                        outcome: SyncOutcome::CheckoutFailed,
                        err: Some(e),
                        duration_ms: op_duration_ms(s_time),
                        ..sync_result
                    }),
                },
            },
            Err(e) => Ok(SyncResult {
                outcome: SyncOutcome::FetchFailed,
                err: Some(e),
                duration_ms: op_duration_ms(s_time),
                ..sync_result
            }),
        },
        Err(e) => Ok(SyncResult {
            outcome: SyncOutcome::SetRemoteFailed,
            err: Some(e),
            duration_ms: op_duration_ms(s_time),
            ..sync_result
        }),
    }
}

pub fn sync(
    manifest: Manifest,
    root_path: PathBuf,
    force: bool,
    fetch_only: bool,
    groups: Option<Vec<String>>,
) -> Result<Vec<SyncResult>, SyncError> {
    // Set up reporting, status, and multithreaded containers.
    let ongoing_syncs: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
    let sync_results: Arc<Mutex<Vec<SyncResult>>> = Arc::new(Mutex::new(Vec::new()));

    let applicable_groups = manifest
        .clone()
        .projects
        .into_par_iter()
        .filter(|p| p.in_groups(&groups));
    let project_count = applicable_groups.clone().count();

    overall_status_thread(
        Arc::clone(&sync_results),
        Arc::clone(&ongoing_syncs),
        project_count,
    );

    applicable_groups.for_each(|p| {
        // Just drain if we're shutting down.
        if is_shutting_down() {
            return;
        }

        let p_path = p.path.clone();

        add_sync(&ongoing_syncs, &p_path);

        // Get the fetch url.
        let remotes = manifest.remotes.clone();
        let remote_name = p.remote.clone().unwrap();
        let remote_name = remote_name.as_str();
        let fetch_url =
            Url::parse(&remotes.get(remote_name).unwrap().fetch.clone().unwrap()).unwrap();

        // Make the actual sync.
        match sync_one(
            p,
            root_path.clone(),
            &fetch_url,
            fetch_only,
            remote_name,
            force,
        ) {
            Ok(sr) => push_result(&sync_results, &sr),
            Err(e) => {
                error!("Unexpected error during sync:\n{:#?}", e)
            }
        }

        rm_sync(&ongoing_syncs, &p_path);
    });

    let ret = sync_results
        .lock()
        .expect("failed to get sync_results mutex")
        .clone();

    // Log the failures.
    for sr in ret
        .clone()
        .into_iter()
        .filter(|s| s.outcome != SyncOutcome::Success)
        .collect::<Vec<_>>()
    {
        error!("{:?}", sr)
    }

    Ok(ret)
}
