# Portable Packages

Route-Switch packages bundle prompt templates, datasets, logs, and analytics into portable, version-controlled directories for migration and disaster recovery.

## Package Contents

A package includes:

| File | Description |
|------|-------------|
| `manifest.yaml` | Template metadata, variables, default routing |
| `package.yaml` | Bundle manifest with metadata |
| `dataset/<template>.db` | Per-prompt SQLite dataset |
| `logs/recent.jsonl` | Recent invocations for inspection |
| `analytics/<file>.duckdb` | Optional DuckDB analytics snapshot |

Every package is automatically initialized as a git repository.

## Exporting Packages

### Basic Export

```bash
./route-switch package export \
  --template-id support-flow \
  --output-dir packages \
  --config config.yaml
```

### With Analytics

Include the analytics snapshot:

```bash
./route-switch package export \
  --template-id support-flow \
  --output-dir packages \
  --include-analytics \
  --config config.yaml
```

### Limit Logs

Control how many recent logs to include:

```bash
./route-switch package export \
  --template-id support-flow \
  --output-dir packages \
  --logs-limit 200 \
  --config config.yaml
```

### Export Options

| Flag | Description |
|------|-------------|
| `--template-id` | Template to export (required) |
| `--output-dir` | Destination directory |
| `--include-analytics` | Include DuckDB snapshot |
| `--logs-limit` | Number of recent logs (default: 100) |

## Package Structure

After export:

```
packages/support-flow-20241208-104500/
├── .git/                    # Git repository
├── manifest.yaml            # Template definition
├── package.yaml             # Bundle metadata
├── dataset/
│   └── support-flow.db      # SQLite dataset
├── logs/
│   └── recent.jsonl         # Recent invocations
└── analytics/
    └── metrics.duckdb       # Analytics snapshot
```

## Importing Packages

### Basic Import

```bash
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml
```

### With Analytics Restore

Restore the analytics database:

```bash
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml \
  --restore-analytics
```

### Overwrite Existing

Replace existing files:

```bash
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml \
  --overwrite
```

### Import Options

| Flag | Description |
|------|-------------|
| `--path` | Package directory (required) |
| `--restore-analytics` | Copy DuckDB to analytics path (default: true) |
| `--overwrite` | Replace existing manifest/dataset/logs |

## After Import

The imported template is available at:

```
data/prompts/<template-id>/
├── manifest.yaml
├── <template-id>.db
└── logs/
    └── recent.jsonl
```

The template is immediately available for:

- Gateway mode
- CLI optimization
- Future exports

## Version Control

Packages are git repositories by default. This means:

- All changes are tracked
- You can inspect history
- Collaborate on prompt development
- Roll back to previous versions

### Viewing History

```bash
cd packages/support-flow-20241208-104500
git log --oneline
```

### Making Changes

```bash
cd packages/support-flow-20241208-104500
# Edit manifest.yaml
git add manifest.yaml
git commit -m "Updated prompt wording"
```

## Migration Workflow

### Export from Source

```bash
./route-switch package export \
  --template-id support-flow \
  --output-dir packages \
  --include-analytics \
  --config config.yaml
```

### Transfer to Destination

```bash
scp -r packages/support-flow-20241208-104500 user@dest:/packages/
```

### Import on Destination

```bash
./route-switch package import \
  --path /packages/support-flow-20241208-104500 \
  --config config.yaml \
  --restore-analytics
```

### Start Gateway

```bash
./route-switch --config config.yaml \
  --gateway \
  --template-id support-flow
```

## Disaster Recovery

Packages serve as backups:

1. **Regular exports** - Schedule periodic exports of critical templates
2. **Off-site storage** - Store packages in cloud storage or remote servers
3. **Quick recovery** - Import and restore in minutes

## Best Practices

1. **Include analytics** - Helps maintain routing decisions
2. **Version package names** - Use timestamps in output directories
3. **Commit before export** - Ensure source templates are committed
4. **Test imports** - Verify imported templates work correctly
5. **Document packages** - Add notes to package.yaml
