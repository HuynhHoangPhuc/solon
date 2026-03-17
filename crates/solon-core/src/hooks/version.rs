/// Version hook: print the solon binary version.
use anyhow::Result;

pub fn run() -> Result<()> {
    println!("{}", env!("CARGO_PKG_VERSION"));
    Ok(())
}
