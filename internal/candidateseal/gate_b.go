package candidateseal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	GateBSchema   = "orquesta.candidate-gate-b-binding.v1"
	GateBStatus   = "prepared_not_executed"
	GateBProtocol = "agentmicrovm.local.v1"
)

var ErrGateBInvalid = errors.New("candidate_gate_b.invalid")

// GateBConnector identifies the exact public connector consumed by Orquesta.
// Physical runtime assets and receipts deliberately remain outside this
// non-physical binding.
type GateBConnector struct {
	Module         string `json:"module"`
	Version        string `json:"version"`
	ModuleSum      string `json:"module_sum"`
	ContractSHA256 string `json:"contract_sha256"`
}

// GateBBinding binds the public connector to one already verified Gate A
// candidate. It can only express NO-GO: physical receipts are still required.
type GateBBinding struct {
	Schema                 string         `json:"schema"`
	Gate                   string         `json:"gate"`
	Status                 string         `json:"status"`
	GateACandidateSHA256   string         `json:"gate_a_candidate_sha256"`
	Protocol               string         `json:"protocol"`
	Connector              GateBConnector `json:"connector"`
	PhysicalReceiptsNeeded bool           `json:"physical_receipts_required"`
	BindingSHA256          string         `json:"binding_sha256"`
}

func BuildGateBBinding(gateA Manifest, connector GateBConnector) (GateBBinding, error) {
	if err := validateManifest(gateA); err != nil || !validGateBConnector(connector) {
		return GateBBinding{}, ErrGateBInvalid
	}
	binding := GateBBinding{
		Schema: GateBSchema, Gate: "B", Status: GateBStatus,
		GateACandidateSHA256: gateA.CandidateSHA256, Protocol: GateBProtocol,
		Connector: connector, PhysicalReceiptsNeeded: true,
	}
	binding.BindingSHA256 = gateBBindingDigest(binding)
	return binding, nil
}

func VerifyGateBBinding(binding GateBBinding, gateA Manifest) error {
	if err := validateManifest(gateA); err != nil || binding.Schema != GateBSchema ||
		binding.Gate != "B" || binding.Status != GateBStatus || binding.Protocol != GateBProtocol ||
		!binding.PhysicalReceiptsNeeded || !validGateBConnector(binding.Connector) ||
		binding.GateACandidateSHA256 != gateA.CandidateSHA256 ||
		binding.BindingSHA256 != gateBBindingDigest(binding) {
		return ErrGateBInvalid
	}
	return nil
}

func EncodeGateBBinding(binding GateBBinding) ([]byte, error) {
	if !validGateBBindingSelf(binding) {
		return nil, ErrGateBInvalid
	}
	encoded, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return nil, ErrGateBInvalid
	}
	return append(encoded, '\n'), nil
}

func DecodeGateBBinding(encoded []byte) (GateBBinding, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var binding GateBBinding
	if err := decoder.Decode(&binding); err != nil {
		return GateBBinding{}, ErrGateBInvalid
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF || !validGateBBindingSelf(binding) {
		return GateBBinding{}, ErrGateBInvalid
	}
	return binding, nil
}

func validGateBBindingSelf(binding GateBBinding) bool {
	return binding.Schema == GateBSchema && binding.Gate == "B" && binding.Status == GateBStatus &&
		binding.Protocol == GateBProtocol && binding.PhysicalReceiptsNeeded &&
		validDigest(binding.GateACandidateSHA256) && validGateBConnector(binding.Connector) &&
		binding.BindingSHA256 == gateBBindingDigest(binding)
}

func validGateBConnector(connector GateBConnector) bool {
	return strings.TrimSpace(connector.Module) == connector.Module && connector.Module != "" &&
		strings.TrimSpace(connector.Version) == connector.Version && connector.Version != "" &&
		strings.TrimSpace(connector.ModuleSum) == connector.ModuleSum && connector.ModuleSum != "" &&
		validDigest(connector.ContractSHA256)
}

func gateBBindingDigest(binding GateBBinding) string {
	binding.BindingSHA256 = ""
	encoded, _ := json.Marshal(binding)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
