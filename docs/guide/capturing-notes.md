# Capturing notes

Create notes from the CLI, the TUI, or through an MCP agent.

## CLI

```bash
# Quick note with inline content
noted add -t "Todo" -c "Buy groceries" -T "personal,todo"

# Open your $EDITOR
noted add -t "Journal entry"

# From a template
noted add -t "Sprint retro" --template meeting
```

See the [commands reference](/reference/commands#adding-notes) for all flags.

## TUI

In the Notes view, press `n` to create a note. Type `[[` in the editor to trigger wikilink
autocomplete.

## MCP

Use the `noted_create` tool:

```json
{
  "title": "Meeting notes",
  "content": "Discussed roadmap.",
  "tags": ["work"]
}
```

## Note links

Use `[[Note title]]` to link to other notes. noted tracks outgoing links and backlinks automatically.

## Templates

Templates support variables such as `{{title}}`, `{{date}}`, `{{time}}`, and `{{datetime}}`.
Read more in [Templates](/guide/templates).
