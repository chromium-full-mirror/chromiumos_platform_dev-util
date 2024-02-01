// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use std::{
    cell::RefCell,
    collections::{HashMap, HashSet},
    env, fmt,
    fs::File,
    io::BufReader,
    iter::FromIterator,
    path::Path,
    rc::Rc,
};

use log::warn;
use xml::{attribute::OwnedAttribute, reader::XmlEvent, EventReader};

#[derive(Clone, Debug, Default)]
pub struct Remote {
    pub name: Option<String>,
    pub fetch: Option<String>,
    pub alias: Option<String>,
    pub review: Option<String>,
}

#[derive(Clone, Debug)]
pub struct RepoHooks {
    pub in_project: Option<String>,
    pub enabled_list: Option<String>,
}

#[derive(Clone, Debug)]
pub struct Manifest {
    pub projects: Vec<Project>,
    pub remotes: HashMap<String, Remote>,
    pub repohooks: Option<RepoHooks>,
}

#[derive(Clone)]
pub struct Project {
    pub path: String,
    pub name: String,
    pub remote: Option<String>,
    pub revision: Option<String>,
    pub groups: HashSet<String>,
    pub annotations: HashMap<String, String>,
    pub copyfile: HashMap<String, String>,
    pub linkfile: HashMap<String, String>,
}

// Here we use std::ptr::eq to compare the *addresses* of `self` and `other`.
impl PartialEq for Project {
    fn eq(&self, other: &Project) -> bool {
        std::ptr::eq(self, other)
    }
}

impl Project {
    // TODO: This doesn't quite meet the repo spec yet. In particular we don't
    // have support for the implicit 'all' or the path: name: prefixes for syncing a
    // single repo. We may or may not decide this is important.
    pub fn in_groups(&self, groups: &Option<Vec<String>>) -> bool {
        match groups {
            Some(g) => !self
                .groups
                .is_disjoint(&HashSet::from_iter(g.iter().cloned())),
            None => !self.groups.contains("notdefault"),
        }
    }
}

impl Default for Project {
    fn default() -> Project {
        Project {
            // 'path' and 'name' are both mandatory.
            path: "uninitialized".to_string(),
            name: "uninitialized".to_string(),
            remote: None,
            revision: None,
            groups: HashSet::new(),
            annotations: HashMap::new(),
            copyfile: HashMap::new(),
            linkfile: HashMap::new(),
        }
    }
}

struct Annotation {
    pub key: Option<String>,
    pub value: Option<String>,
}

impl Annotation {
    fn default() -> Annotation {
        Annotation {
            key: None,
            value: None,
        }
    }
}

impl fmt::Debug for Project {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Project")
            .field("path", &self.path)
            .field("name", &self.name)
            .field("remote", &self.remote)
            .field("revision", &self.revision)
            .field("groups", &self.groups)
            .field("annotations", &self.annotations)
            .field("copyfile", &self.copyfile)
            .finish()
    }
}

/// Merges two manifests into the first using one as the primary in all conflicts.
fn merge_manifest(one: &mut Manifest, two: Manifest) {
    one.projects
        .extend(two.projects.iter().map(|x| x.to_owned()));
    one.repohooks = one.repohooks.clone().or(two.repohooks);
    for (k, v) in two.remotes {
        one.remotes.insert(k, v);
    }
}

pub fn parse_manifest(file_name: &Path) -> Manifest {
    let file = File::open(file_name)
        .unwrap_or_else(|_| panic!("cannot open manifest {}", file_name.display()));
    let file = BufReader::new(file);
    let parser = EventReader::new(file);

    let mut projects: Vec<Project> = Vec::new();
    let mut remotes: HashMap<String, Remote> = HashMap::new();
    let mut sub_manifests: Vec<Manifest> = Vec::new();
    let mut repohooks = Option::None;
    let mut default_revision = Option::None;
    let mut default_remote = Option::None;

    // Keep track of the current project for nested element additions (e.g. annotations).
    let mut proj = Rc::new(RefCell::new(Project::default()));

    // Search a vec in linear time for the key.
    fn find_attr<'a>(key: &'a str) -> impl Fn(&&'a OwnedAttribute) -> bool {
        move |i: &&'a OwnedAttribute| -> bool { i.name.local_name == key }
    }

    fn get_attr(attrs: &[OwnedAttribute], key: &str) -> Option<String> {
        attrs.iter().find(find_attr(key)).map(|x| x.value.clone())
    }

    // Set cwd to the manifest's dir as paths to includes are relative.
    let old_cwd = env::current_dir();
    let manifest_dir = Path::new(file_name)
        .parent()
        .expect("manifest location is not a valid");

    // If it's a null string or otherwise doesn't exist, stay in place.
    if manifest_dir.exists() {
        let manifest_dir = manifest_dir
            .canonicalize()
            .expect("unable to canonicalize path");
        env::set_current_dir(manifest_dir).expect("error setting cwd");
    }

    for e in parser {
        match e {
            Ok(XmlEvent::StartElement {
                name, attributes, ..
            }) => match name.local_name.as_str() {
                "project" => {
                    let name = get_attr(&attributes, "name").expect("cannot get name attribute");
                    let mut p = Project {
                        path: get_attr(&attributes, "path").unwrap_or(name.clone()),
                        name,
                        remote: get_attr(&attributes, "remote"),
                        revision: get_attr(&attributes, "revision"),
                        groups: HashSet::new(),
                        annotations: HashMap::new(),
                        copyfile: HashMap::new(),
                        linkfile: HashMap::new(),
                    };
                    if let Some(group_str) = get_attr(&attributes, "groups").or(None) {
                        p.groups = group_str.split(',').map(|s| s.to_string()).collect();
                    }
                    proj = Rc::new(RefCell::new(p));
                }
                "annotation" => {
                    let mut this_annotation = Annotation::default();
                    for a in attributes {
                        match a.name.local_name.as_str() {
                            "name" => {
                                this_annotation.key = Some(a.value);
                            }
                            "value" => {
                                this_annotation.value = Some(a.value);
                            }
                            _ => {
                                warn!("Warning! unknown annotation attribute: {:?}", a.name);
                            }
                        }
                    }
                    if this_annotation.key.is_none() || this_annotation.value.is_none() {
                        panic!("Malformed annotation");
                    }
                    proj.borrow_mut().annotations.insert(
                        this_annotation.key.expect("missing key"),
                        this_annotation.value.expect("missing value"),
                    );
                }
                "remote" => {
                    let remote_name = get_attr(&attributes, "name").expect("unnamed remote");
                    remotes.insert(
                        remote_name.clone(),
                        Remote {
                            name: Some(remote_name.clone()),
                            fetch: get_attr(&attributes, "fetch"),
                            alias: get_attr(&attributes, "alias"),
                            review: get_attr(&attributes, "review"),
                        },
                    );
                }
                "include" => {
                    sub_manifests.push(parse_manifest(Path::new(
                        &get_attr(&attributes, "name")
                            .expect("missing include name")
                            .as_str(),
                    )));
                }
                "default" => {
                    default_revision = get_attr(&attributes, "revision");
                    default_remote = get_attr(&attributes, "remote");
                }
                "manifest" => {
                    // Cover this case but it's the root node so noop.
                }
                "notice" => {
                    // Cover this case but it's a noop.
                }
                "repo-hooks" => {
                    repohooks = Some(RepoHooks {
                        in_project: Some(
                            get_attr(&attributes, "in-project").expect("missing in-project"),
                        ),
                        enabled_list: Some(
                            get_attr(&attributes, "enabled-list").expect("enabled-list"),
                        ),
                    });
                }
                "copyfile" => {
                    proj.borrow_mut().copyfile.insert(
                        get_attr(&attributes, "src").expect("missing src"),
                        get_attr(&attributes, "dest").expect("missing dest"),
                    );
                }
                "linkfile" => {
                    proj.borrow_mut().copyfile.insert(
                        get_attr(&attributes, "src").expect("missing src"),
                        get_attr(&attributes, "dest").expect("missing dest"),
                    );
                }
                _ => {
                    panic!("unparsed tag {:?}", name)
                }
            },
            Ok(XmlEvent::EndElement { name, .. }) => {
                if name.local_name.as_str() == "project" {
                    projects.push(proj.borrow().clone());
                }
            }
            Err(e) => {
                panic!("bad manifest: {:?}", e);
            }
            _ => {}
        }
    }

    let _ = env::set_current_dir(old_cwd.expect("error resetting cwd"));

    let mut ret_manifest = Manifest {
        projects,
        remotes,
        repohooks,
    };
    for sub_manifest in sub_manifests {
        merge_manifest(&mut ret_manifest, sub_manifest)
    }

    // Defaults should be thrown away and replicated in every project.
    let mut new_projs: Vec<Project> = Vec::new();
    for mut p in ret_manifest.projects {
        match p.remote {
            Some(_) => {}
            None => p.remote = default_remote.clone(),
        }
        match p.revision {
            Some(_) => {}
            None => p.revision = default_revision.clone(),
        }
        new_projs.push(p.clone());
    }
    ret_manifest.projects = new_projs;

    ret_manifest
}
