-- B10.5d4b0: la aplicación conoce el requisito antes de abrir el efecto
-- físico. Puede fijarlo exactamente al preparar queued -> dispatching.
-- La transición histórica dispatching -> running sigue admitida para
-- snapshots anteriores. El requisito nunca puede revertirse ni aparecer en
-- otra transición.
DROP TRIGGER executions_environment_preservation_write_once;

CREATE TRIGGER executions_environment_preservation_write_once
BEFORE UPDATE OF environment_preservation_required ON executions
WHEN NOT (
  NEW.environment_preservation_required = OLD.environment_preservation_required
  OR (
    OLD.state = 'queued'
    AND NEW.state = 'dispatching'
    AND OLD.environment_preservation_required = 0
    AND NEW.environment_preservation_required = 1
  )
  OR (
    OLD.state = 'dispatching'
    AND NEW.state = 'running'
    AND OLD.environment_preservation_required = 0
    AND NEW.environment_preservation_required = 1
  )
)
BEGIN
  SELECT RAISE(ABORT, 'sqlite.execution_environment_preservation_write_once');
END;
