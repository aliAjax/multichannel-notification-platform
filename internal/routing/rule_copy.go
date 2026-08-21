package routing

func cloneRule(r Rule) Rule {
	matches := make([]Match, len(r.Matches))
	copy(matches, r.Matches)
	providers := make([]string, len(r.ProviderNames))
	copy(providers, r.ProviderNames)
	r.Matches = matches
	r.ProviderNames = providers
	return r
}
