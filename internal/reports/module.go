package reports

type Module struct {
	Report Protocol
	// Contract file extensions beside .yaml and .yml, lowercase and with the dot.
	Extensions []string
}

func protocolFor(reportKey string, modules []Module) (Protocol, error) {
	for _, module := range modules {
		if module.Report.ReportKey() == reportKey {
			return module.Report, nil
		}
	}

	return nil, newUnknownProtocolError(reportKey)
}
