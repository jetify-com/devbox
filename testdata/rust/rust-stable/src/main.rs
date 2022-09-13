use rustc_version::{version, version_meta, Channel};
use semver::{Version};

fn main() {
    let installed_version = version().unwrap();
    println!("Installed version is {}", installed_version);

    // Ensure we do not go backwards
    let min_stable_version = "1.63.0";
    if installed_version < Version::parse(min_stable_version).unwrap() {
        panic!("Version {} is less than expected version of {}", installed_version, min_stable_version)
    }

    match version_meta().unwrap().channel {
        Channel::Stable => (), // do nothing
        ch @ _ => panic!("Expected Stable channel but got channel: {}", channel_as_str(ch)),
    }
}

// Alas Channel doesn't implement the Display trait so we do this.
fn channel_as_str(ch: Channel) -> &'static str {
    return match ch {
        Channel::Dev => "dev",
        Channel::Nightly => "nightly",
        Channel::Beta => "beta",
        Channel::Stable => "stable",
    }
}
    

