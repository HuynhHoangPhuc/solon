/// Skill management CLI commands: create, validate, catalog.
use anyhow::{anyhow, Result};
use clap::{Args, Subcommand};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Args)]
pub struct SkillArgs {
    #[command(subcommand)]
    pub command: SkillCommand,
}

#[derive(Subcommand)]
pub enum SkillCommand {
    /// Scaffold a new skill directory with template SKILL.md
    Create(CreateArgs),
    /// Validate a SKILL.md file for correct structure
    Validate(ValidateArgs),
    /// List all installed skills across plugins
    Catalog(CatalogArgs),
}

#[derive(Args)]
pub struct CreateArgs {
    /// Skill name (without sl: prefix)
    pub name: String,
    /// Plugin directory to create skill in
    #[arg(long, default_value = "plugins/solon-core")]
    pub plugin: String,
}

#[derive(Args)]
pub struct ValidateArgs {
    /// Path to SKILL.md file or skill directory
    pub path: String,
}

#[derive(Args)]
pub struct CatalogArgs {
    /// Specific plugin directory to list (default: all plugins/)
    #[arg(long)]
    pub plugin: Option<String>,
    /// Output format: json or text
    #[arg(long, default_value = "text")]
    pub format: String,
}

pub fn run(args: SkillArgs) -> Result<()> {
    match args.command {
        SkillCommand::Create(a) => run_create(a),
        SkillCommand::Validate(a) => run_validate(a),
        SkillCommand::Catalog(a) => run_catalog(a),
    }
}

// --- Create ---

const SKILL_TEMPLATE: &str = r#"---
name: sl:{name}
description: "{description}"
argument-hint: "[args]"
---

# {title}

## Usage

```
/sl:{name} [args]
```

## Workflow

1. Step one
2. Step two

## References

- `references/` — Additional context files
"#;

fn run_create(args: CreateArgs) -> Result<()> {
    // Reject path traversal in skill name
    if args.name.contains("..") || args.name.contains('/') || args.name.contains('\\') {
        return Err(anyhow!(
            "Invalid skill name: must not contain path separators or '..'"
        ));
    }

    let skill_dir = PathBuf::from(&args.plugin).join("skills").join(&args.name);

    if skill_dir.exists() {
        return Err(anyhow!(
            "Skill directory already exists: {}",
            skill_dir.display()
        ));
    }

    fs::create_dir_all(skill_dir.join("references"))?;
    fs::create_dir_all(skill_dir.join("scripts"))?;

    let title = args
        .name
        .split('-')
        .map(|w| {
            let mut c = w.chars();
            match c.next() {
                None => String::new(),
                Some(f) => f.to_uppercase().to_string() + c.as_str(),
            }
        })
        .collect::<Vec<_>>()
        .join(" ");

    let content = SKILL_TEMPLATE
        .replace("{name}", &args.name)
        .replace("{title}", &title)
        .replace("{description}", &format!("TODO: describe {}", &args.name));

    fs::write(skill_dir.join("SKILL.md"), content)?;

    println!("Created skill scaffold:");
    println!("  {}/", skill_dir.display());
    println!("  ├── SKILL.md");
    println!("  ├── references/");
    println!("  └── scripts/");
    Ok(())
}

// --- Validate ---

fn run_validate(args: ValidateArgs) -> Result<()> {
    let path = PathBuf::from(&args.path);
    let skill_md = if path.is_dir() {
        path.join("SKILL.md")
    } else {
        path.clone()
    };

    if !skill_md.exists() {
        return Err(anyhow!("File not found: {}", skill_md.display()));
    }

    let content = fs::read_to_string(&skill_md)?;
    let mut errors: Vec<String> = Vec::new();

    // Check YAML frontmatter
    if let Some(rest) = content.strip_prefix("---\n") {
        if let Some(end) = rest.find("\n---") {
            let frontmatter = &rest[..end];
            if !frontmatter.contains("name:") {
                errors.push("Frontmatter missing required field: name".into());
            } else if !frontmatter.contains("sl:") {
                errors.push("Skill name must use sl: prefix".into());
            }
            if !frontmatter.contains("description:") {
                errors.push("Frontmatter missing required field: description".into());
            }
        } else {
            errors.push("Malformed YAML frontmatter (missing closing ---)".into());
        }
    } else {
        errors.push("Missing YAML frontmatter (must start with ---)".into());
    }

    // Check file size
    let line_count = content.lines().count();
    if line_count > 500 {
        errors.push(format!(
            "SKILL.md is {} lines (recommended max: 500)",
            line_count
        ));
    }

    // Check for stale CKE references using word-boundary-aware regex
    let ck_re = regex::Regex::new(r"\bck:[a-z][a-z0-9-]*").unwrap();
    if ck_re.is_match(&content) {
        errors.push("Contains stale CKE references (ck:) — remap to sl:".into());
    }

    if errors.is_empty() {
        println!("Valid: {}", skill_md.display());
        Ok(())
    } else {
        println!("Invalid: {}", skill_md.display());
        for e in &errors {
            println!("  - {}", e);
        }
        Err(anyhow!("{} validation error(s) found", errors.len()))
    }
}

// --- Catalog ---

#[derive(serde::Serialize)]
struct SkillEntry {
    plugin: String,
    name: String,
    description: String,
}

fn run_catalog(args: CatalogArgs) -> Result<()> {
    let plugins_dir = PathBuf::from("plugins");
    let plugin_dirs: Vec<PathBuf> = if let Some(ref p) = args.plugin {
        vec![PathBuf::from(p)]
    } else if plugins_dir.is_dir() {
        fs::read_dir(&plugins_dir)?
            .filter_map(|e| e.ok())
            .map(|e| e.path())
            .filter(|p| p.is_dir())
            .collect()
    } else {
        return Err(anyhow!("No plugins/ directory found"));
    };

    let mut entries: Vec<SkillEntry> = Vec::new();

    for plugin_dir in &plugin_dirs {
        let skills_dir = plugin_dir.join("skills");
        if !skills_dir.is_dir() {
            continue;
        }
        let plugin_name = plugin_dir
            .file_name()
            .unwrap_or_default()
            .to_string_lossy()
            .to_string();

        let mut skill_dirs: Vec<_> = fs::read_dir(&skills_dir)?
            .filter_map(|e| e.ok())
            .map(|e| e.path())
            .filter(|p| p.is_dir())
            .collect();
        skill_dirs.sort();

        for skill_dir in skill_dirs {
            let skill_md = skill_dir.join("SKILL.md");
            if !skill_md.exists() {
                continue;
            }
            let (name, desc) = parse_skill_frontmatter(&skill_md);
            entries.push(SkillEntry {
                plugin: plugin_name.clone(),
                name,
                description: desc,
            });
        }
    }

    if args.format == "json" {
        println!("{}", serde_json::to_string_pretty(&entries)?);
        return Ok(());
    }

    // Text output grouped by plugin
    let mut current_plugin = String::new();
    for e in &entries {
        if e.plugin != current_plugin {
            if !current_plugin.is_empty() {
                println!();
            }
            let count = entries.iter().filter(|x| x.plugin == e.plugin).count();
            println!("Plugin: {} ({} skills)", e.plugin, count);
            current_plugin = e.plugin.clone();
        }
        println!("  {:30} {}", e.name, e.description);
    }

    if entries.is_empty() {
        println!("No skills found.");
    }

    Ok(())
}

fn parse_skill_frontmatter(path: &Path) -> (String, String) {
    let content = match fs::read_to_string(path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("  warn: cannot read {}: {}", path.display(), e);
            return ("unknown".into(), format!("(unreadable: {})", e));
        }
    };
    let mut name = String::from("unknown");
    let mut description = String::new();

    if let Some(rest) = content.strip_prefix("---\n") {
        if let Some(end) = rest.find("\n---") {
            let fm = &rest[..end];
            for line in fm.lines() {
                let line = line.trim();
                if let Some(val) = line.strip_prefix("name:") {
                    name = val.trim().trim_matches('"').to_string();
                } else if let Some(val) = line.strip_prefix("description:") {
                    description = val.trim().trim_matches('"').to_string();
                }
            }
        }
    }
    (name, description)
}
