# Semantic search

noted can search notes by meaning using veclite and Ollama embeddings.

## Prerequisites

1. [Ollama](https://ollama.ai) running locally
2. The default embedding model pulled:

```bash
ollama pull nomic-embed-text
```

## Configuration

```bash
export NOTED_VECLITE_PATH=~/.local/share/noted/vectors.veclite
export NOTED_EMBEDDING_MODEL=nomic-embed-text
```

## Sync notes

```bash
noted sync
noted sync --force
```

## Use semantic search

```bash
noted recall "user preferences" --semantic
noted forget --query "temporary" --force
```

Agents can use `noted_semantic_search` and `noted_recall` with semantic mode.

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NOTED_VECLITE_PATH` | Path to veclite database | (disabled) |
| `NOTED_EMBEDDING_MODEL` | Ollama model | `nomic-embed-text` |
| `OLLAMA_HOST` | Ollama server URL | `http://localhost:11434` |
