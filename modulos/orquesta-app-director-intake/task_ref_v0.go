package orquestaappdirectorintake

func appDirectorTaskRefPrefixV0(spec AppDirectorInputSpecV0) string {
	return safeDirectorIntakeRefPartV0(firstDirectorIntakeValueV0(spec.SpecID, spec.App.Slug))
}
