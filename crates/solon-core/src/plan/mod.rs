pub mod naming;
pub mod resolver;
pub mod scaffolder;
pub mod validator;

pub use naming::{build_plan_dir_name, format_date, sanitize_slug};
pub use resolver::{
    enrich_resolve_result, extract_frontmatter_field, resolve_plan_path, ResolveResult,
};
pub use scaffolder::{scaffold_plan, ScaffoldMode, ScaffoldResult};
pub use validator::{validate_plan, ValidationResult, ValidationStats};
