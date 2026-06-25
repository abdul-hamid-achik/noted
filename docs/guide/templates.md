# Templates

Templates are reusable note blueprints with variable interpolation.

## Create a template

```bash
noted template create -n "meeting" -c "# {{title}}

Date: {{date}}

## Attendees

## Notes

## Action Items
- [ ] "
```

## Variables

| Variable | Replaced with |
|----------|---------------|
| `{{date}}` | Current date (YYYY-MM-DD) |
| `{{time}}` | Current time (HH:MM) |
| `{{datetime}}` | Current date and time |
| `{{title}}` | Note title |

## Use a template

```bash
noted add -t "Sprint retro" --template meeting
```

## TUI

Open the Templates view with `7` to create a note from a template.

## Manage templates

```bash
noted template list
noted template show meeting
noted template edit meeting
noted template delete meeting
```
