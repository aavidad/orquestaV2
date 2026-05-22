package orquestarunqueue

import "testing"

func TestNormalizeRunQueuePriorityCommandV0CompactaRefs(t *testing.T) {
	got := NormalizeRunQueuePriorityCommandV0(RunQueuePriorityCommandV0{
		RunRef:       " run-ref-001 ",
		QueueRef:     " global ",
		AppRef:       " app-ref-001 ",
		Status:       " closed ",
		RequestedBy:  " director ",
		EvidenceRefs: []string{" evidence-1 ", "evidence-1", ""},
	})

	if got.RunRef != "run-ref-001" ||
		got.QueueRef != "global" ||
		got.AppRef != "app-ref-001" ||
		got.Status != "closed" ||
		got.RequestedBy != "director" ||
		len(got.EvidenceRefs) != 1 {
		t.Fatalf("command=%+v", got)
	}
}

func TestValidateRunQueuePriorityCommandV0ExigeRunRef(t *testing.T) {
	issues := ValidateRunQueuePriorityCommandV0(RunQueuePriorityCommandV0{})
	if len(issues) != 1 || issues[0].Field != "run_ref" {
		t.Fatalf("issues=%+v", issues)
	}
}
