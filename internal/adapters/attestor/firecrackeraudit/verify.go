//go:build linux

package firecrackeraudit

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

type Verification struct {
	Events         uint64
	HeadDigest     string
	LastTransition Transition
	Pending        bool
	PendingRef     string
	PendingSubject string
}

type digestRecord struct {
	SchemaVersion uint32     `json:"schema_version"`
	Seq           uint64     `json:"seq"`
	PrevDigest    string     `json:"prev_digest"`
	StreamRef     string     `json:"stream_ref"`
	OperationRef  string     `json:"operation_ref"`
	SubjectRef    string     `json:"subject_ref"`
	Transition    Transition `json:"transition"`
	Diagnostic    Diagnostic `json:"diagnostic"`
}

func recordDigest(record Record) (string, error) {
	canonical, err := json.Marshal(digestRecord{
		SchemaVersion: record.SchemaVersion,
		Seq:           record.Seq,
		PrevDigest:    record.PrevDigest,
		StreamRef:     record.StreamRef,
		OperationRef:  record.OperationRef,
		SubjectRef:    record.SubjectRef,
		Transition:    record.Transition,
		Diagnostic:    record.Diagnostic,
	})
	if err != nil {
		return "", auditError(CodeEvent, err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func encodeRecord(record Record, maxEventBytes int) ([]byte, error) {
	digest, err := recordDigest(record)
	if err != nil {
		return nil, err
	}
	record.EventDigest = digest
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, auditError(CodeEvent, err)
	}
	encoded = append(encoded, '\n')
	if len(encoded) > maxEventBytes {
		return nil, auditError(CodeLimit, nil)
	}
	return encoded, nil
}

// Verify checks exact canonical JSONL, sequence, transition pairs and every
// digest link. It never repairs or truncates input.
func Verify(reader io.Reader, streamRef string, maxEventBytes int, maxEvents uint64) (Verification, error) {
	if !validHex(streamRef) {
		return Verification{}, auditError(CodeRef, nil)
	}
	if reader == nil || maxEventBytes <= 0 || maxEvents == 0 {
		return Verification{}, auditError(CodeConfig, nil)
	}
	if maxEvents > uint64((1<<63-2)/int64(maxEventBytes)) {
		return Verification{}, auditError(CodeConfig, nil)
	}
	limit := int64(maxEventBytes)*int64(maxEvents) + 1
	content, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return Verification{}, auditError(CodeIO, err)
	}
	if int64(len(content)) == limit {
		return Verification{}, auditError(CodeLimit, nil)
	}
	if len(content) > 0 && content[len(content)-1] != '\n' {
		return Verification{}, auditError(CodePartial, nil)
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 4096), maxEventBytes)
	verification := Verification{HeadDigest: EmptyDigest}
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if len(line)+1 > maxEventBytes || verification.Events == maxEvents {
			return Verification{}, auditError(CodeLimit, nil)
		}
		var record Record
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return Verification{}, auditError(CodeIntegrity, err)
		}
		if decoder.Decode(&struct{}{}) != io.EOF {
			return Verification{}, auditError(CodeIntegrity, nil)
		}
		canonical, err := json.Marshal(record)
		if err != nil || !bytes.Equal(canonical, line) {
			return Verification{}, auditError(CodeIntegrity, err)
		}
		if record.SchemaVersion != SchemaVersion || record.Seq != verification.Events+1 ||
			record.StreamRef != streamRef || record.PrevDigest != verification.HeadDigest ||
			!validHex(record.EventDigest) {
			return Verification{}, auditError(CodeIntegrity, nil)
		}
		event := Event{
			OperationRef: record.OperationRef,
			SubjectRef:   record.SubjectRef,
			Transition:   record.Transition,
			Diagnostic:   record.Diagnostic,
		}
		if err := validateEvent(event); err != nil {
			return Verification{}, auditError(CodeIntegrity, err)
		}
		expected, err := recordDigest(record)
		if err != nil || expected != record.EventDigest {
			return Verification{}, auditError(CodeIntegrity, err)
		}
		if verification.Pending {
			if record.Transition == Started || record.OperationRef != verification.PendingRef ||
				record.SubjectRef != verification.PendingSubject {
				return Verification{}, auditError(CodeTransition, nil)
			}
			verification.Pending = false
			verification.PendingRef = ""
			verification.PendingSubject = ""
		} else {
			if record.Transition != Started {
				return Verification{}, auditError(CodeTransition, nil)
			}
			verification.Pending = true
			verification.PendingRef = record.OperationRef
			verification.PendingSubject = record.SubjectRef
		}
		verification.Events = record.Seq
		verification.HeadDigest = record.EventDigest
		verification.LastTransition = record.Transition
	}
	if err := scanner.Err(); err != nil {
		if err == bufio.ErrTooLong {
			return Verification{}, auditError(CodeLimit, err)
		}
		return Verification{}, auditError(CodeIO, err)
	}
	return verification, nil
}
