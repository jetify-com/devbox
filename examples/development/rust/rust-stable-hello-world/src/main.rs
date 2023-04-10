fn main() {
  println!("{}", hello_world())
}

fn hello_world() -> &'static str {
    return "Hello, world!";
}

mod tests {
  use super::*;

  #[test]
  fn test_hello_world() {
    assert_eq!(hello_world(), "Hello, world!")
  }
}
