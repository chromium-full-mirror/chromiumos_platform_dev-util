// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use std::{fs::File, io::Read, sync::RwLock};

use git2::{Cred, CredentialType, ErrorClass, ErrorCode};
use log::{debug, error};
use url::Url;

use crate::error::AuthError;

#[derive(Clone, Debug)]
pub struct CookieEntry {
    domain: String,
    _sub_domains: bool,
    _path: String,
    _https_only: bool,
    _expiration: u64,
    _name: String,
    value: String,
}

// Cookies, in reverse order they were found in the file.
static COOKIE_JAR: RwLock<Option<Vec<CookieEntry>>> = RwLock::new(None);

static DEFAULT_COOKIE_FILE: &str = "~/.gitcookies";

// Netscape, blastin' from the pastin'. Load up the cookie jar.
pub fn initialize_gitcookies(cookie_path: Option<String>) -> Result<(), AuthError> {
    // Get a lock on the cookie jar, initialize it.
    let mut jar: std::sync::RwLockWriteGuard<'_, Option<Vec<CookieEntry>>> =
        COOKIE_JAR.write().expect("write lock on cookie jar failed");

    if jar.is_some() {
        error!("You should not call load_gitcookies twice, will not reload");
        return Ok(());
    }

    *jar = Some(Vec::new());
    let jar = jar
        .as_mut()
        .expect("could not get mutable cookie jar reference");

    // Look in the user's home dir.
    let cookie_path = cookie_path.unwrap_or(DEFAULT_COOKIE_FILE.to_string());
    let gitcookies = shellexpand::tilde(&cookie_path).into_owned();
    let file = File::open(gitcookies);
    let mut file = match file {
        Ok(f) => f,
        Err(err) => {
            if cookie_path != DEFAULT_COOKIE_FILE {
                return Err(AuthError::MissingCookies(format!("{}", err)));
            } else {
                return Ok(());
            }
        }
    };

    let mut contents = String::new();
    file.read_to_string(&mut contents).unwrap();

    contents.split('\n').for_each(|l| {
        let mut fields = l.split('\t').collect::<Vec<&str>>();
        fields.iter_mut().for_each(|x| {
            *x = x.trim();
        });

        if fields.len() == 7 && !fields[0].starts_with('#') {
            jar.push(CookieEntry {
                domain: fields[0]
                    .to_ascii_lowercase()
                    .parse()
                    .expect("cookie field domain invalid"),
                _sub_domains: fields[1]
                    .to_ascii_lowercase()
                    .parse()
                    .expect("cookie field sub domain invalid"),
                _path: fields[2].parse().expect("cookie field path invalid"),
                _https_only: fields[3]
                    .to_ascii_lowercase()
                    .parse()
                    .expect("cookie https_only invalid"),
                _expiration: fields[4].parse().expect("cookie field expiration invalid"),
                _name: fields[5].parse().expect("cookie field name invalid"),
                value: fields[6].parse().expect("cookie field value invalid"),
            })
        }
    });
    jar.reverse();
    debug!(".gitcookies was read and contained {} values", jar.len());

    Ok(())
}

// Given to libgit2 as a callback when we're asked for auth.
// Only supports USER_PASS_PLAINTEXT from .gitcookies.
pub fn auth_callback(
    url: &str,
    _username_from_url: Option<&str>,
    allowed_types: CredentialType,
) -> Result<Cred, git2::Error> {
    match allowed_types {
        CredentialType::USER_PASS_PLAINTEXT => {
            // From the bottom up find the first domain match. This is assuming
            // tools append to cookies but don't remove existing entries.
            let domain = String::from(Url::parse(url).unwrap().domain().unwrap());
            let jar = COOKIE_JAR.read().expect("could not get lock on cookie jar");
            let jar = jar.as_ref().expect("cookie jar not initialized");
            match jar.iter().find(|x| x.domain == domain) {
                Some(c) => {
                    // Example value: git-engeg.google.com=1/xxizOLvvl7ZfT3vOHJJn3GA1m70WpjQ-u47lwZIH31I (and no this isn't a good cred.)
                    let c = c.value.split('=').collect::<Vec<&str>>();
                    match c.len() {
                        2 => Cred::userpass_plaintext(c[0], c[1]),
                        _ => Err(git2::Error::new(
                            ErrorCode::Auth,
                            ErrorClass::Callback,
                            "invalid gitcookie entry (expected user=pass)",
                        )),
                    }
                }
                None => Err(git2::Error::new(
                    ErrorCode::Auth,
                    ErrorClass::Callback,
                    format!("no cookie for domain: {}", domain),
                )),
            }
        }
        _ => Err(git2::Error::new(
            ErrorCode::Auth,
            ErrorClass::Callback,
            format!("auth method {:?} not implemented", allowed_types),
        )),
    }
}
