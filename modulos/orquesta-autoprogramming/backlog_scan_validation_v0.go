package orquestaautoprogramming

import "strings"

func autoprogrammingRequestBacklogScanIssuesV0(
	scan AutoprogrammingBacklogScanV0,
) []AutoprogrammingRequestIssueV0 {
	if strings.TrimSpace(scan.Epoch) == "" &&
		len(scan.Documents) == 0 &&
		len(scan.ReservationRefs) == 0 {
		return nil
	}
	var issues []AutoprogrammingRequestIssueV0
	if strings.TrimSpace(scan.Epoch) == "" {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"backlog_scan_epoch_missing",
			"backlog_scan.epoch",
			"epoch requerido para scanner de backlog",
		))
	}
	if len(compactStringsV0(scan.ReservationRefs)) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"backlog_scan_reservation_missing",
			"backlog_scan.reservation_refs",
			"reserva de write-set documental requerida",
		))
	}
	if len(scan.Documents) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"backlog_scan_docs_missing",
			"backlog_scan.documents",
			"hash/linea de documentos requeridos",
		))
	}
	for _, doc := range scan.Documents {
		issues = append(issues, autoprogrammingBacklogScanDocIssuesV0(doc)...)
	}
	return issues
}

func autoprogrammingBacklogScanDocIssuesV0(
	doc AutoprogrammingBacklogDocumentV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if !autoprogrammingRequestWriteSetPathAllowedV0(doc.Path) {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"backlog_scan_doc_path_invalid",
			"backlog_scan.documents.path",
			"ruta de documento no permitida: "+doc.Path,
		))
	}
	if strings.TrimSpace(doc.SHA256) == "" {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"backlog_scan_doc_hash_missing",
			"backlog_scan.documents.sha256",
			"hash de documento requerido",
		))
	}
	return issues
}
