package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	historicalRegistrySourceSHA256    = "sha256:56e48ffa327e9b593628ca07cc025ed073ad28cb529d25d11dde4a7440b94033"
	historicalRegistryDefinitionCount = 25
)

// historicalRegistryDefinitionDigests accredits the additive transition from
// the historical global registry identity to definition-scoped identities.
// An unchanged historical definition keeps its persisted global digest so the
// audit store can preserve exact replay. Any semantic change falls through to
// a new definition digest and therefore conflicts with the persisted admission.
var historicalRegistryDefinitionDigests = map[string]string{
	"orquesta.goals.create@1":                "sha256:6a94d3577e9261625c651f580cdb61389c8bf2c55cee825704097763dc3caf52",
	"orquesta.goals.amend@1":                 "sha256:e9c0c00394eccb1ae2d30f3efba172532cb106fe7977659cc39180a7bf1fa65b",
	"orquesta.goals.get@1":                   "sha256:90c486e9a124b57d690131ac03df0273aeb468067809ceb4967f7b239fadc809",
	"orquesta.goals.list@1":                  "sha256:79dfe4fd6b54d79bd08d84fa2a02c4c181092cc537093cc4188f9ca733ab06a8",
	"orquesta.artifacts.read@1":              "sha256:de15b47de20aa8751b845046938caac9c8d5a5528de8fb93983ed905a561201a",
	"orquesta.system.status@1":               "sha256:7503067e9937b867d6c10db7d9e04da823f535e4f461f26433886ade55dab95b",
	"orquesta.projects.memberships.grant@1":  "sha256:a10fba53fd7e537a2e1debb9f767e9d5eaccc7ad5a5c7a99af3f7bec2ac40ccc",
	"orquesta.projects.memberships.revoke@1": "sha256:128b567887bb33f0f9e05c6e82894f1403673d2210bb70709adfa3275363b4c0",
	"orquesta.director.claim@1":              "sha256:db7ed5637eb5bddb78ddaafd87e1320754dbc0d027552ef7d333a66060d41ebf",
	"orquesta.director.renew@1":              "sha256:8294368716518f8e734d2b8ca1f183451b589fd703c651e63e53078455b9d91d",
	"orquesta.director.plan.propose@1":       "sha256:288b6b2f409aa2514389cf0f4bb0df15f305ff46a5eacd73ed6211adac17bf9c",
	"orquesta.goals.control@1":               "sha256:c985fa9859388ace42f87186cd8cedc859a06be33c97c971bc288c9a6f48333e",
	"orquesta.effects.decide@1":              "sha256:6171ea829c2286dd28edb0f9ce8987594844265eb428a7378368e6da9dfe2ac7",
	"orquesta.changes.list@1":                "sha256:57fc562f43552a60db553a3a9b7d7e1d79635eaded00ee2b74a258cec451b145",
	"orquesta.changes.integrate@1":           "sha256:a44a662f570325623cd8bbfd228262eba8b3764da6232875c810372a8c846475",
	"orquesta.mailbox.admit@1":               "sha256:58a57b47e39589bb2a4bd5214aefc501990874298af363bee1273efdeabb9f26",
	"orquesta.mailbox.claim@1":               "sha256:4960b5d2508ec8e08cf0bee433c503d9ae062faff76f0cbf9026a42640ce6fcb",
	"orquesta.mailbox.mark_delivered@1":      "sha256:5b290a32a2166f749137486eb5cd2d4b6c14a299f034908058e192ae8b1e8006",
	"orquesta.mailbox.consume@1":             "sha256:4d958bc6998b7d9a9b3d3e783dabec173aaac9e6c16563e352846604f9169813",
	"orquesta.mailbox.get@1":                 "sha256:d08519345ac1d12cd972ccc6c403aca5bceb941faf5570fa54e359c0d6055ab7",
	"orquesta.mailbox.list@1":                "sha256:388f43ed77b5e4aabcb791c6f523a24151c4c917449aad24ddd921ed2445834c",
	"orquesta.mailbox.acknowledge@1":         "sha256:6c2e1160803e196ddd3e52f9d186261d70faaea7e061246168685438fce07ea7",
	"orquesta.mailbox.block@1":               "sha256:716035cf22c9268d4c5f6c36cc9e1947aba53a98a29300415f5f700794b57bf9",
	"orquesta.council.round.open@1":          "sha256:c8f0f540ecad8289303ec1efd3aa94e979b86e088e758628d4ded23037c1a820",
	"orquesta.council.skip@1":                "sha256:2bb738964ec2e1cf48c3ab6a67587b8d7ae35ff187fe0b97f89d77fa08665b26",
}

func registryAdmissionDigest(definition Definition) string {
	definitionDigest := definitionIdentityDigest(definition)
	if historicalRegistryDefinitionDigests[definition.ID+"@"+definition.Version] == definitionDigest {
		return historicalRegistrySourceSHA256
	}
	return definitionDigest
}

func definitionIdentityDigest(definition Definition) string {
	encoded, err := json.Marshal(definition)
	if err != nil {
		return ""
	}
	var value any
	if err := decodeSingle(encoded, &value, true); err != nil {
		return ""
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}
