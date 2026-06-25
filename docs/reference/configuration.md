# Configuration

noted follows the XDG Base Directory Specification.

## Default paths

| Path | Description |
|------|-------------|
| `~/.local/share/noted/noted.db` | SQLite database (index) |
| `~/.local/share/noted/vault` | Markdown vault |
| `~/.local/share/noted/vectors.veclite` | Vector database (optional) |

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `EDITOR` | Editor for composing notes | `nvim` |
| `NOTED_VAULT` | Markdown vault directory | `~/.local/share/noted/vault` |
| `NOTED_VECLITE_PATH` | Path to veclite database | (disabled) |
| `NOTED_EMBEDDING_MODEL` | Ollama embedding model | `nomic-embed-text` |
| `OLLAMA_HOST` | Ollama server URL | `http://localhost:11434` |

## CLI overrides

Pass `--db` or `--vault` to any command:

```bash
noted --db /tmp/demo.db list
noted --vault ~/Documents/noted-vault add -t "Idea"
```
