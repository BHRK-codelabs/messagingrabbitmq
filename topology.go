package rabbitmq

import "github.com/BHRK-codelabs/corekit/configkit"

func NewTopologyNames(cfg *configkit.Config, spec TopologySpec) TopologyNames {
	return BuildTopologyNames(
		NamingConvention{
			Prefix:  cfg.RabbitMQ.NamespacePrefix,
			Env:     cfg.App.Environment.String(),
			Domain:  cfg.RabbitMQ.DomainName,
			Version: cfg.RabbitMQ.Version,
		},
		spec,
	)
}

