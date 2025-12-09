# Alias System Redesign Proposals

## Current Issues

The current `.mpp.txt` alias system has several pain points:

1. **Long, repetitive lines**: Each alias is a single long line with all options concatenated
2. **Poor readability**: Hard to understand complex aliases at a glance
3. **Difficult maintenance**: Copy-paste between similar aliases leads to errors and inconsistency
4. **No reusability**: Cannot compose aliases from common building blocks
5. **Limited expressiveness**: Cannot define variables, shared patterns, or conditional logic
6. **No validation**: Easy to make syntax errors that only appear at runtime

### Example of Current Format

```
js_dev: --role-message "You are a JavaScript expert" -i src/**/*.js -i src/**/*.ts -e **/__tests__/* -e **/*.test.js
go_dev: --role-message "You are a Go expert" -i **/*.go -e **/*_test.go
python_dev: --role-message "You are a Python expert" -i **/*.py -e **/tests/** --extra-context "Focus on PEP 8 compliance and best practices"
```

Notice the repetition of patterns and the difficulty in seeing what's different between similar aliases.

---

## Design Proposal 1: YAML-Based Structured Configuration

### Overview
Replace `.mpp.txt` with `.mpp.yaml` or `.mpp.yml` files that use YAML's structure to organize aliases with variables, inheritance, and reusable components.

### Format Example

```yaml
# .mpp.yaml

# Define reusable variables
variables:
  expert_roles:
    javascript: "You are a JavaScript expert"
    go: "You are a Go expert"
    python: "You are a Python expert"
  
  common_excludes:
    tests: ["**/__tests__/*", "**/*.test.js", "**/*_test.go", "**/tests/**"]
    build: ["**/node_modules/**", "**/vendor/**", "**/__pycache__/**", "**/dist/**", "**/build/**"]
    all: ["${tests}", "${build}"]

  file_patterns:
    javascript: ["**/*.js", "**/*.jsx", "**/*.ts", "**/*.tsx"]
    go: ["**/*.go"]
    python: ["**/*.py"]

# Define aliases
aliases:
  js_dev:
    role_message: "${variables.expert_roles.javascript}"
    include:
      - "src/**/*.js"
      - "src/**/*.ts"
    exclude: "${variables.common_excludes.tests}"
  
  go_dev:
    role_message: "${variables.expert_roles.go}"
    include: "${variables.file_patterns.go}"
    exclude:
      - "**/*_test.go"
  
  python_dev:
    role_message: "${variables.expert_roles.python}"
    include: "${variables.file_patterns.python}"
    exclude: "${variables.common_excludes.tests}"
    extra_context: "Focus on PEP 8 compliance and best practices"
  
  code_review:
    role_message: "You are a senior code reviewer"
    extra_context: "Look for bugs, security issues, and code quality problems"
    last_words: "Provide specific, actionable feedback"
  
  # Compose from other aliases
  full_js_review:
    extends: ["js_dev", "code_review"]
    # Additional overrides
    exclude: "${variables.common_excludes.all}"
```

### Advantages
- **Clear structure**: YAML's hierarchy makes relationships obvious
- **Reusability**: Variables eliminate repetition
- **Composition**: `extends` allows combining aliases
- **Validation**: YAML parsers provide syntax validation
- **Tooling**: Many editors have excellent YAML support
- **Readability**: Much easier to understand complex configurations

### Disadvantages
- **Breaking change**: Requires migration from `.mpp.txt`
- **Complexity**: More features to implement and maintain
- **Dependency**: Requires YAML parsing library

### Implementation Notes
- Support both `.mpp.txt` (legacy) and `.mpp.yaml` with YAML taking precedence
- Provide migration tool: `mpp migrate-config` to convert old format
- Use Go library like `gopkg.in/yaml.v3` for parsing

---

## Design Proposal 2: Template-Based Aliases with Parameterization

### Overview
Extend the current simple format to support templates and parameters, allowing aliases to be functions that accept arguments.

### Format Example

```
# .mpp.txt

# Define templates (aliases that accept parameters)
# Syntax: template_name(param1, param2, ...): options using ${param1}, ${param2}
template lang_expert(lang, ext, exclude_pattern): --role-message "You are a ${lang} expert" -i **/*.${ext} -e ${exclude_pattern}

# Define variables for reuse
@var test_excludes="**/__tests__/* **/*.test.js **/*_test.go **/tests/**"
@var common_context="Focus on code quality and best practices"

# Use templates to define concrete aliases
js_dev: lang_expert("JavaScript", "js", "${test_excludes}")
go_dev: lang_expert("Go", "go", "**/*_test.go")
python_dev: lang_expert("Python", "py", "**/tests/**") --extra-context "${common_context}"

# Standard aliases still work
code_review: --role-message "You are a senior code reviewer" --extra-context "Look for bugs and security issues"

# Compose aliases by referencing others
full_review: ${code_review} -i **/*.go -i **/*.py -i **/*.js
```

### Usage Example

```bash
# Use a template directly with parameters
mpp -a 'lang_expert("Rust", "rs", "**/target/**")' -q "Review this code"

# Use a predefined alias
mpp -a js_dev -q "Explain this function"
```

### Advantages
- **Backward compatible**: Extends current format rather than replacing it
- **Flexible**: Parameters allow one template to serve many use cases
- **Familiar**: Variable substitution syntax is common (${var})
- **Composable**: Can reference other aliases
- **Simple migration**: Old aliases work as-is

### Disadvantages
- **Complex parsing**: Requires more sophisticated parser than current line-based approach
- **Learning curve**: Users need to understand template syntax
- **Error handling**: Parameter type checking and validation needed

### Implementation Notes
- Extend current config parser to handle `@var` and `template` keywords
- Support both direct template invocation and predefined aliases
- Validate parameter counts and types at parse time

---

## Design Proposal 3: Hierarchical Configuration with Inheritance

### Overview
Introduce a hierarchical alias system where aliases can inherit from and override parent aliases, similar to CSS or object-oriented inheritance.

### Format Example

```
# .mpp.txt

# Base aliases define common patterns
[base.expert]
role_message: "You are an expert"

[base.exclude_tests]
exclude: **/__tests__/*
exclude: **/*.test.*
exclude: **/tests/**

[base.common_includes]
include: src/**/*
include: lib/**/*

# Language-specific aliases inherit and extend
[js_dev] extends: base.expert, base.exclude_tests
role_message: "You are a JavaScript expert"  # Override
include: **/*.js
include: **/*.ts
exclude: **/__tests__/*  # Adds to inherited excludes

[go_dev] extends: base.expert, base.exclude_tests
role_message: "You are a Go expert"
include: **/*.go
exclude: **/*_test.go  # Adds to inherited excludes

[python_dev] extends: base.expert, base.exclude_tests
role_message: "You are a Python expert"
include: **/*.py
extra_context: "Focus on PEP 8 compliance"

# Multiple inheritance
[full_js_review] extends: js_dev, base.common_includes
extra_context: "Look for bugs and security issues"
last_words: "Provide actionable feedback"

# Can also reference other non-base aliases
[production_js] extends: js_dev
exclude: **/__mocks__/*  # Additional exclusions
exclude: **/fixtures/**
```

### Advantages
- **DRY principle**: Share common configuration across aliases
- **Clear relationships**: Inheritance makes patterns explicit
- **Incremental definition**: Build complex aliases from simple parts
- **Flexible**: Multiple inheritance for maximum reusability
- **Familiar concept**: Inheritance is well-understood by developers

### Disadvantages
- **Complexity**: Inheritance chains can be hard to follow
- **Merge semantics**: Need clear rules for how multi-valued fields (include, exclude) combine
- **New syntax**: Square brackets and `extends` keyword differ from current format
- **Migration effort**: Existing files need restructuring

### Implementation Notes
- Use INI-like section syntax `[alias_name]` for compatibility with config parsers
- Define clear merge rules: includes/excludes accumulate, single-value fields override
- Resolve inheritance at load time to flatten before use
- Detect circular inheritance and report errors

---

## Design Proposal 4: JSON/TOML with Schema Validation

### Overview
Use JSON or TOML format with JSON Schema validation to provide IDE autocomplete, validation, and documentation.

### TOML Format Example

```toml
# .mpp.toml

# Shared definitions
[shared.roles]
javascript = "You are a JavaScript expert"
go = "You are a Go expert"
python = "You are a Python expert"

[shared.patterns]
js = ["**/*.js", "**/*.ts", "**/*.jsx", "**/*.tsx"]
go = ["**/*.go"]
py = ["**/*.py"]

[shared.excludes]
tests = ["**/__tests__/*", "**/*.test.*", "**/tests/**"]
builds = ["**/node_modules/**", "**/dist/**", "**/build/**"]

# Alias definitions
[[alias]]
name = "js_dev"
role_message = "You are a JavaScript expert"
include = ["src/**/*.js", "src/**/*.ts"]
exclude = ["**/__tests__/*", "**/*.test.js"]

[[alias]]
name = "go_dev"
role_message = "You are a Go expert"
include = ["**/*.go"]
exclude = ["**/*_test.go"]

[[alias]]
name = "python_dev"
role_message = "You are a Python expert"
include = ["**/*.py"]
exclude = ["**/tests/**"]
extra_context = "Focus on PEP 8 compliance and best practices"

[[alias]]
name = "code_review"
role_message = "You are a senior code reviewer"
extra_context = "Look for bugs, security issues, and code quality problems"
last_words = "Provide specific, actionable feedback"

# Reference shared definitions
[[alias]]
name = "full_stack"
role_message = "You are a full-stack developer"
include = [
  "**/*.js", "**/*.ts",  # JavaScript/TypeScript
  "**/*.py",             # Python
  "**/*.go"              # Go
]
exclude = [
  "**/__tests__/*",
  "**/*.test.*",
  "**/tests/**",
  "**/node_modules/**",
  "**/dist/**"
]
```

### JSON Schema for Validation

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "MPP Configuration",
  "type": "object",
  "properties": {
    "shared": {
      "type": "object",
      "properties": {
        "roles": {"type": "object"},
        "patterns": {"type": "object"},
        "excludes": {"type": "object"}
      }
    },
    "alias": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "role_message": {"type": "string"},
          "include": {"type": "array", "items": {"type": "string"}},
          "exclude": {"type": "array", "items": {"type": "string"}},
          "force_include": {"type": "array", "items": {"type": "string"}},
          "extra_context": {"type": "string"},
          "last_words": {"type": "string"},
          "question": {"type": "string"}
        },
        "required": ["name"]
      }
    }
  }
}
```

### Advantages
- **IDE Support**: JSON Schema enables autocomplete and validation in modern editors
- **Type Safety**: Schema validates structure at load time
- **Documentation**: Schema serves as API documentation
- **Standard Format**: TOML/JSON are widely used and understood
- **Tooling**: Excellent existing tools for editing and validating

### Disadvantages
- **Verbosity**: TOML/JSON more verbose than current simple format
- **No native references**: Sharing between aliases requires manual duplication or tool support
- **Migration**: Complete format change requires conversion
- **Learning**: Users need to understand TOML/JSON syntax

### Implementation Notes
- Provide `.mpp-schema.json` for IDE integration
- Support both TOML and JSON (`.mpp.toml` and `.mpp.json`)
- Generate schema from code for type safety
- Provide migration command: `mpp migrate-config --to toml`

---

## Design Proposal 5: Modular Include System

### Overview
Allow aliases to include/import other alias files, enabling a modular organization where teams can share common aliases while customizing locally.

### Format Example

```
# team-aliases.mpp.txt (shared team file)
@include "~/.mpp/base-aliases.txt"
@include "https://example.com/company-standards.mpp.txt"  # Remote includes

# Team-wide standards
standard_review: --role-message "You are a code reviewer" --extra-context "Check company coding standards"
js_team: --role-message "You are a JavaScript expert" -i src/**/*.js -e **/__tests__/*

# .mpp.txt (local project file)
@include "./team-aliases.mpp.txt"

# Override or extend team aliases
js_team: ${js_team} -i lib/**/*.js  # Extend team alias

# Project-specific aliases
project_specific: -i custom/**/*.go -q "Review this custom code"
```

### Advantages
- **Modularity**: Separate concerns into different files
- **Sharing**: Teams can distribute standard aliases
- **Flexibility**: Projects can override or extend shared aliases
- **Remote includes**: Load aliases from URLs for company-wide standards
- **Gradual adoption**: Add includes without changing existing aliases

### Disadvantages
- **Complexity**: Include resolution adds complexity
- **Performance**: Multiple file reads and HTTP requests
- **Security**: Remote includes pose security risks
- **Circular deps**: Need to detect and prevent circular includes

### Implementation Notes
- Support file paths relative to config file location
- Support `~/` for user home directory
- Support `http://` and `https://` for remote includes
- Cache remote includes with TTL
- Detect circular includes and report error
- Process includes depth-first for predictable override behavior

---

## Recommendation

**Recommended Approach: Hybrid of Proposals 1 and 5**

Implement **Proposal 1 (YAML-based)** as the primary new format with **Proposal 5 (Modular includes)** added for sharing.

### Rationale

1. **YAML** provides the best balance of:
   - Readability and maintainability
   - Powerful features (variables, composition)
   - Existing tooling and familiarity
   - Validation capabilities

2. **Modular includes** add:
   - Team collaboration capabilities
   - Separation of concerns
   - Gradual migration path

3. **Migration path**:
   - Phase 1: Support both `.mpp.txt` (current) and `.mpp.yaml` (new)
   - Phase 2: Add `@include` directive to both formats
   - Phase 3: Provide `mpp migrate-config` command
   - Phase 4: Deprecate `.mpp.txt` in documentation (keep support for backward compatibility)

### Implementation Priority

1. **Short term** (next release):
   - Implement basic YAML support (Proposal 1)
   - Keep `.mpp.txt` for backward compatibility
   - Add migration tool

2. **Medium term** (following release):
   - Add variables and inheritance in YAML
   - Implement `@include` directive
   - Add schema validation

3. **Long term**:
   - Consider adding template parameters (Proposal 2) if needed
   - Evaluate remote includes security and caching
   - Build web-based alias editor/validator

---

## Alternative Considerations

### Proposal 2 (Template-based)
Best for users who want maximum flexibility without changing file format. Could be combined with any other proposal.

### Proposal 3 (Hierarchical)
Best for teams with many similar aliases. More complex to understand but very powerful for DRY.

### Proposal 4 (JSON/TOML with Schema)
Best for teams who prioritize IDE integration and validation. TOML could replace YAML in Proposal 1.

### Proposal 5 (Modular)
Essential for any solution that aims to support team collaboration. Should be added regardless of primary format choice.

---

## Migration Strategy

Regardless of chosen design, provide:

1. **Backward compatibility**: Continue supporting `.mpp.txt`
2. **Migration tool**: `mpp migrate-config --from txt --to yaml`
3. **Validation tool**: `mpp validate-config` to check syntax
4. **Documentation**: Clear examples and migration guide
5. **Gradual rollout**: Announce deprecation well in advance
6. **Error messages**: Helpful messages pointing to migration guide

---

## Appendix: Feature Comparison Matrix

| Feature | Proposal 1 (YAML) | Proposal 2 (Template) | Proposal 3 (Hierarchical) | Proposal 4 (TOML/JSON) | Proposal 5 (Modular) |
|---------|-------------------|----------------------|---------------------------|------------------------|---------------------|
| Variables/Reuse | ✓✓✓ | ✓✓ | ✓ | ✓ | - |
| Composition | ✓✓✓ | ✓✓ | ✓✓✓ | ✓ | ✓✓✓ |
| Readability | ✓✓✓ | ✓✓ | ✓✓ | ✓✓ | ✓ |
| IDE Support | ✓✓ | ✓ | ✓ | ✓✓✓ | ✓ |
| Validation | ✓✓✓ | ✓✓ | ✓✓ | ✓✓✓ | ✓ |
| Backward Compatible | ✓ | ✓✓✓ | - | - | ✓✓ |
| Implementation Complexity | ✓✓ | ✓ | ✓ | ✓✓ | ✓✓ |
| Team Collaboration | ✓ | ✓ | ✓ | ✓ | ✓✓✓ |

Legend: ✓✓✓ Excellent, ✓✓ Good, ✓ Fair, - Poor/Not applicable
