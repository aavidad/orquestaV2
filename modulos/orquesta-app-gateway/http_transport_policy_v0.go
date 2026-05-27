package orquestaappgateway

const (
	GatewayHTTPTransportProfileInternalInProcessV0 = "internal_inprocess"
	GatewayHTTPProxyPolicyDenyV0                   = "deny"
)

type GatewayHTTPTransportPolicyV0 struct {
	Profile     string
	ProxyPolicy string
	Network     string
}

func DefaultGatewayHTTPTransportPolicyV0() GatewayHTTPTransportPolicyV0 {
	return GatewayHTTPTransportPolicyV0{
		Profile:     GatewayHTTPTransportProfileInternalInProcessV0,
		ProxyPolicy: GatewayHTTPProxyPolicyDenyV0,
		Network:     "none",
	}
}
