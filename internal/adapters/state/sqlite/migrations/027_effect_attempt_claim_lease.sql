-- B10.5d1b: conserva la concesión histórica exacta del intento físico.
-- Los intentos anteriores a V27 solo pueden mantener NULL cuando una prueba
-- causal terminal hace innecesaria su reconciliación.
ALTER TABLE effect_attempts
 ADD COLUMN claim_lease_until INTEGER
 CHECK (claim_lease_until IS NULL OR claim_lease_until > started_at);

-- El backfill Go se ejecuta dentro de esta misma transacción. La guarda
-- inmutable se retira únicamente mientras completa las filas derivables.
DROP TRIGGER effect_attempts_immutable_update;

-- orquesta:go-backfill effect_attempt_claim_lease

CREATE TRIGGER effect_attempts_claim_lease_guard
BEFORE INSERT ON effect_attempts
WHEN NEW.claim_lease_until IS NULL
  OR NEW.claim_lease_until <= NEW.started_at
  OR NOT EXISTS (
    SELECT 1 FROM outbox action
    WHERE action.ref = NEW.action_ref
      AND action.fence = NEW.action_fence
      AND action.claimed_by = NEW.worker_ref
      AND action.claimed_until = NEW.claim_lease_until
      AND action.claim_token IS NOT NULL
      AND NEW.ref = 'effect-attempt:' || action.ref || ':' || action.claim_token
  )
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_claim_lease_invalid'); END;

CREATE TRIGGER effect_attempts_immutable_update BEFORE UPDATE ON effect_attempts
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_immutable'); END;
