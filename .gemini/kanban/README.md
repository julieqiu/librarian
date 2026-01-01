# Librarian Kanban Board

This directory implements the "Filesystem as State" pattern for managing complex Librarian workflows.

## Workflow

1.  **TODO:** New tasks are created as JSON files in `todo/`.
2.  **IN PROGRESS:** When an agent starts a task, the file is moved to `in_progress/`.
3.  **DONE:** Successfully completed tasks are moved to `done/` with full logs.
4.  **BLOCKED:** Tasks requiring human intervention or external dependencies are moved to `blocked/`.

## Ticket Schema

```json
{
  "id": "TASK-ID",
  "title": "Short description",
  "persona": "toolmaker | orchestrator | rustacean | gopher | pythonista | scribe | critic",
  "status": "todo | in_progress | done | blocked",
  "description": "Detailed instructions",
  "context": {
    "files": [],
    "apis": [],
    "repos": []
  },
  "history": [
    {
      "timestamp": "ISO-8601",
      "action": "moved to in_progress",
      "note": "Starting execution"
    }
  ]
}
```
