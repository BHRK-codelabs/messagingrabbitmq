package rabbitmq

import "fmt"

type TopologyNames struct {
	ExchangeKind    string
	ExchangeName    string
	DLQExchangeName string
	Routes          map[string]RouteNames
}

type RouteNames struct {
	RoutingKey string
	QueueName  string
	DLQ        DLQNames
}

type DLQNames struct {
	RoutingKey string
	QueueName  string
}

type NamingConvention struct {
	Prefix  string
	Env     string
	Domain  string
	Version string
}

type RouteSpec struct {
	MessageType string
	Action      string
	RoutingKey  string
}

type TopologySpec struct {
	ExchangeKind string
	Routes       map[string]RouteSpec
}

func BuildTopologyNames(convention NamingConvention, spec TopologySpec) TopologyNames {
	if convention.Env == "" {
		convention.Env = "dev"
	}

	exchangeBase := joinSegments(convention.Prefix, convention.Env, convention.Domain)
	routes := make(map[string]RouteNames, len(spec.Routes))
	for name, route := range spec.Routes {
		routes[name] = newRouteNames(buildRoutingKey(convention, route, spec.ExchangeKind))
	}

	exchangeKind := normalizeExchangeKind(spec.ExchangeKind)
	return TopologyNames{
		ExchangeKind:    exchangeKind,
		ExchangeName:    exchangeBase + ".exchange",
		DLQExchangeName: exchangeBase + ".dlq.exchange",
		Routes:          routes,
	}
}

func (t TopologyNames) Route(name string) (RouteNames, bool) {
	route, ok := t.Routes[name]
	return route, ok
}

func buildRoutingKey(convention NamingConvention, route RouteSpec, exchangeKind string) string {
	if route.RoutingKey != "" {
		return route.RoutingKey
	}
	if normalizeExchangeKind(exchangeKind) == "fanout" {
		return ""
	}
	return joinSegments(
		convention.Prefix,
		convention.Env,
		convention.Domain,
		convention.Version,
		route.MessageType,
		route.Action,
	)
}

func newRouteNames(routingKey string) RouteNames {
	return RouteNames{
		RoutingKey: routingKey,
		QueueName:  routingKey + ".queue",
		DLQ: DLQNames{
			RoutingKey: routingKey + ".dlq",
			QueueName:  routingKey + ".dlq.queue",
		},
	}
}

func normalizeExchangeKind(kind string) string {
	switch kind {
	case "direct", "fanout", "headers":
		return kind
	default:
		return "topic"
	}
}

func joinSegments(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return fmt.Sprintf("%s", join(filtered, "."))
}

func join(parts []string, separator string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += separator + parts[i]
	}
	return result
}

