// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use std::{
    collections::HashMap,
    fs::{self, create_dir_all, File},
    io::Write,
    path::{Path, PathBuf},
};

use crate::{
    error::{RootError, SnagError, SyncError},
    manifest::{Manifest, Project, Remote},
    sync::{sync, SyncResult},
};

use anyhow::Error;
use log::{debug, error};
use serde::{Deserialize, Serialize};
use url::Url;

const SNAG_ROOT_DIRNAME: &str = ".snag";
const SNAG_ROOT_INFO_FNAME: &str = "root_info.json";
const SNAG_ROOT_MANIFEST_DIR: &str = "manifest";

#[derive(Debug, Deserialize, Serialize)]
pub struct SnagRoot {
    // The path to base of the checkouts.
    pub path: String,

    // The manifest file to use.
    pub manifest: String,

    // The URL to the remote repo containing the manifest .xmls.
    manifest_url: String,

    // The ref to track upstream.
    tracking_ref: String,
}

impl SnagRoot {
    pub fn sync_root(&self) -> Result<SyncResult, Error> {
        let checkout_path = self.manifest_path();
        let fetch_url = Url::parse(&self.manifest_url).expect("Could not parse manifest url");
        let name = fetch_url.path_segments().unwrap().last().unwrap();
        let p = self.get_manifest_project(checkout_path.clone(), name);
        let projects = Vec::from([p]);
        let mut remotes = HashMap::new();
        remotes.insert(
            String::from("cros"),
            Remote {
                name: Some("cros".to_string()),
                fetch: Some(self.manifest_url.to_string()),
                alias: None,
                review: None,
            },
        );

        let m = Manifest {
            projects,
            remotes,
            repohooks: None,
        };
        match sync(m, checkout_path, false, false, None) {
            Ok(r) => match r.first() {
                Some(r) => Ok(r.clone()),
                None => Err(SyncError::Fetch(String::from("failed to clone manifest")).into()),
            },
            Err(e) => Err(e.into()),
        }
    }

    pub fn manifest_path(&self) -> PathBuf {
        Path::new(&self.path)
            .join(SNAG_ROOT_DIRNAME)
            .join(SNAG_ROOT_MANIFEST_DIR)
    }

    /// Gets a project struct customized for the manifest checkout.
    fn get_manifest_project(&self, checkout_path: PathBuf, name: &str) -> Project {
        Project {
            name: name.to_string(),
            path: checkout_path.as_os_str().to_str().unwrap().to_string(),
            revision: Some(self.tracking_ref.clone()),
            remote: Some(String::from("cros")),
            ..Default::default()
        }
    }
}

pub fn discover_root(dir: &str) -> Option<String> {
    let dir = Path::new(dir);
    let dir = match dir.canonicalize() {
        Ok(d) => d,
        Err(_) => return None,
    };
    let cnts = fs::read_dir(dir.to_str().expect("path is invalid")).unwrap();
    for f in cnts {
        let f_name = f.unwrap().file_name();
        if f_name == SNAG_ROOT_DIRNAME {
            let root_info = dir.join(SNAG_ROOT_DIRNAME).join(SNAG_ROOT_INFO_FNAME);
            match root_info.exists() {
                true => return Some(root_info.to_str().unwrap().to_string()),
                false => return None,
            }
        }
    }
    match dir.parent() {
        Some(d) => discover_root(d.to_str().unwrap()),
        None => None,
    }
}

pub fn load_root(d: &str) -> Result<SnagRoot, Error> {
    let root_info = fs::read_to_string(d);
    match root_info {
        Ok(root_info) => match serde_json::from_str(&root_info) {
            Ok(root_info) => {
                debug!("Snag root:\n{:#?}", root_info);
                Ok(root_info)
            }
            Err(e) => Err(RootError::Deserialization(format!("{:#?}", e)).into()),
        },
        Err(e) => {
            error!("Could not read snag root info, raw contents:\n{:#?}", d);
            Err(RootError::Deserialization(format!("{:#?}", e)).into())
        }
    }
}

pub fn create_root(
    path: &str,
    manifest_url: &Url,
    tracking_ref: String,
    clobber: bool,
    manifest: String,
) -> Result<SnagRoot, Error> {
    let existing_root = discover_root(path);
    if existing_root.is_some() {
        if clobber {
            return Err(SnagError::NotImplemented.into());
        } else {
            return Err(RootError::Exists(path.to_string()).into());
        }
    };

    let snag_dir = Path::new(path).join(Path::new(SNAG_ROOT_DIRNAME));
    let snag_f = snag_dir.join(SNAG_ROOT_INFO_FNAME);
    match create_dir_all(snag_dir) {
        Ok(_) => {
            let default_root_info = SnagRoot {
                path: path.to_string(),
                manifest_url: manifest_url.to_string(),
                tracking_ref,
                manifest,
            };

            let json = serde_json::to_string_pretty(&default_root_info)?;
            let mut file = File::create(snag_f)?;
            file.write_all(json.as_bytes())?;
            Ok(default_root_info)
        }
        Err(e) => Err(RootError::Create(format!("{:#?}", e)).into()),
    }
}
