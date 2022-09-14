use rustc_version::{version};
use semver::{Version};

fn main() {
    let installed_version = version().unwrap();
    println!("Installed version is {}", installed_version);

    let expected_version = "1.62.0";
    if installed_version != Version::parse(expected_version).unwrap() {
        panic!("Expected version {} but got installed version: {}", expected_version, installed_version)
    }
}
