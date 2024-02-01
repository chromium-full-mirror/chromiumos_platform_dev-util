// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Custom error types for the snag utility.

use std::fmt::Debug;

#[derive(thiserror::Error, Debug, Clone)]
pub enum Error {}

#[derive(thiserror::Error, Debug, Clone)]
#[error["An error relating to auth has occured"]]
pub enum AuthError {
    #[error("Can not read cookie_path and it was provided: `{0}`")]
    MissingCookies(String),
}

#[derive(thiserror::Error, Debug, Clone)]
#[error["An error relating to the state of the root has occured"]]
pub enum RootError {
    #[error("Root was not able to be created: `{0}`")]
    Create(String),

    #[error("Root into was not able to be deserialized: `{0}`")]
    Deserialization(String),

    #[error("Root already exists: `{0}`")]
    Exists(String),

    #[error("Root was not found")]
    NotFound,
}

#[derive(thiserror::Error, Debug, Clone)]
#[error["An general error has occured"]]

pub enum SnagError {
    #[error("This command is not implemented")]
    NotImplemented,
}

#[derive(thiserror::Error, Debug, Clone, PartialEq)]
#[error["An error syncing a repository has occured"]]
pub enum SyncError {
    #[error("Initialization failed: `{0}`")]
    OpenOrInit(String),

    #[error("Fetch failed: `{0}`")]
    Fetch(String),

    #[error("Setting remote failed: `{0}`")]
    SetRemote(String),

    #[error("Checkout failed: `{0}`")]
    Checkout(String),
}
