-- Convert any existing milestone/gate tasks to regular tasks.
UPDATE task SET type = 'task' WHERE type IN ('milestone', 'gate');
