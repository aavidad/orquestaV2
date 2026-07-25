package firecrackerattestor

import (
	"errors"
	"strconv"
	"strings"
)

func marshalReceipt(receipt Receipt) ([]byte, error) {
	if receipt.Schema != ReceiptSchema || receipt.Status != "passed" ||
		!validDigest(receipt.EvidenceSHA256) ||
		!validDigest(receipt.ConfigSHA256) ||
		!validDigest(receipt.UnitSHA256) ||
		!validDigest(receipt.PrimitivesUnitSHA256) ||
		!validDigest(receipt.LauncherSHA256) ||
		!validDigest(receipt.AssetDigest) ||
		!validDigest(receipt.PolicyDigest) ||
		receipt.E2ESuite != Suite ||
		receipt.MaxConcurrentRuns != ExpectedConcurrentRuns ||
		receipt.PhysicalMicroVMCount != ExpectedConcurrentRuns ||
		receipt.ConcurrentHighWater != ExpectedConcurrentRuns ||
		!receipt.AllAttestationsValid || !receipt.ZeroResidualRuns ||
		!receipt.NetworkAbsent || !receipt.MemorySwapMaxZero ||
		!receipt.APIAbsent || !receipt.VsockAbsent || !receipt.SerialAbsent {
		return nil, errors.New("firecracker_attestor_e2e.receipt_invalid")
	}
	lines := []string{
		"schema=" + receipt.Schema,
		"status=" + receipt.Status,
		"config_sha256=" + receipt.ConfigSHA256,
		"unit_sha256=" + receipt.UnitSHA256,
		"primitives_unit_sha256=" + receipt.PrimitivesUnitSHA256,
		"launcher_sha256=" + receipt.LauncherSHA256,
		"asset_digest=" + receipt.AssetDigest,
		"evidence_sha256=" + receipt.EvidenceSHA256,
		"policy_digest=" + receipt.PolicyDigest,
		"e2e_suite=" + receipt.E2ESuite,
		"max_concurrent_runs=" + strconv.FormatUint(uint64(receipt.MaxConcurrentRuns), 10),
		"physical_microvm_count=" + strconv.FormatUint(uint64(receipt.PhysicalMicroVMCount), 10),
		"concurrent_high_water=" + strconv.FormatUint(uint64(receipt.ConcurrentHighWater), 10),
		"all_attestations_valid=true",
		"zero_residual_runs=true",
		"network_absent=true",
		"api_absent=true",
		"vsock_absent=true",
		"serial_absent=true",
		"memory_swap_max_zero=true",
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}
