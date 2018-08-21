// +build !darwin

package openvpn

// NewClient creates openvpn client with given config params
func NewClient(openvpnBinary string, config *ClientConfig, stateHandler state.Callback, statsHandler bytescount.SessionStatsHandler, credentialsProvider auth.CredentialsProvider) Process {

	return CreateNewProcess(
		openvpnBinary,
		config.GenericConfig,
		state.NewMiddleware(stateCallback),
		bytescount.NewMiddleware(statsHandler, 1*time.Second),
		auth.NewMiddleware(credentialsProvider),
	)
}
