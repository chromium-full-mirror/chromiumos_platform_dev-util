// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

use rand::{thread_rng, Rng};
use std::collections::HashMap;
use std::fs::File;
use std::io::Write;
use std::path::PathBuf;

use tiny_http;

const SERVER_PORT: i32 = 9986;

static SIZES_MB: &[u32; 3] = &[5, 20, 100];

fn main() -> std::result::Result<(), std::io::Error> {
    let mut rng = thread_rng();
    let mut data = [0u8; 1024];
    let mut routes = HashMap::new();

    for size in SIZES_MB {
        let mut p = PathBuf::from("/tmp");
        let filename = format!("{}MB.bin", size);
        p.push(format!("{}MB.bin", size));
        println!("Writing {}...", p.to_string_lossy());

        let mut f = File::create(p.clone())?;
        for _ in 0..size * 1024 {
            rng.fill(&mut data);
            f.write_all(&data)?;
        }
        f.flush()?;

        routes.insert(format!("/{}", filename), p);
    }

    println!("Serving requests");
    let server = tiny_http::Server::http(format!("[::]:{}", SERVER_PORT)).unwrap();
    loop {
        let request = match server.recv() {
            Ok(rq) => rq,
            Err(e) => {
                println!("error: {}", e);
                break;
            }
        };

        let res = match routes.get(request.url()) {
            Some(file) => match File::open(file) {
                Ok(f) => request.respond(tiny_http::Response::from_file(f)),
                Err(e) => {
                    println!("error opening {}: {}", request.url(), e);
                    request.respond(tiny_http::Response::empty(500))
                }
            },
            None => request.respond(tiny_http::Response::empty(404)),
        };

        if let Err(e) = res {
            println!("could not serve request: {}", e);
        }
    }

    println!("Done");
    Ok(())
}
