package orquestadomainworksql

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

func domainWorkSQLRequestFingerprintV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) (string, error) {
	identity, err := orquestadomainwork.BuildDomainWorkJobIdentityV0(request)
	if err != nil {
		return "", err
	}
	return identity.Fingerprint, nil
}
