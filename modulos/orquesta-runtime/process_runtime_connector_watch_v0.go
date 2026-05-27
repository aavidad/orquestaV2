package orquestaruntime

import "time"

func (c *ProcessRuntimeConnectorV0) waitProcessV0(record *processRuntimeRecordV0) {
	_ = record.cmd.Wait()

	c.mu.Lock()
	record.status = ProcessRuntimeStoppedV0
	c.mu.Unlock()
	close(record.done)
}

func (c *ProcessRuntimeConnectorV0) watchAdoptedProcessV0(record *processRuntimeRecordV0) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if processRuntimeProcessAliveV0(record.process) {
			continue
		}
		c.mu.Lock()
		if record.status != ProcessRuntimeStoppedV0 {
			record.status = ProcessRuntimeStoppedV0
		}
		c.mu.Unlock()
		close(record.done)
		return
	}
}
