package orquestastatefileoutbox

import (
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

var _ orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.PendingOutboxReaderPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.OutboxDispatchClaimerPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.OutboxDispatchAckPortV0 = (*FileOutboxLedgerV0)(nil)
var _ orquestaoutboxdispatch.OutboxDispatchAckObservationPortV0 = (*FileOutboxLedgerV0)(nil)
