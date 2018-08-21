// +build darwin

package openvpn

import (
	"github.com/MysteriumNetwork/openvpn3-go-bindings/openvpn3"
	log "github.com/cihub/seelog"
	"strings"
)

const openvpn3SessionPrefx = "[openvpn3 session] "

// NewClient creates openvpn client with given config params
func NewClient(openvpnBinary string, config *ClientConfig, stateHandler Callback, statsHandler SessionStatsHandler, credentialsProvider CredentialsProvider) Process {
	return &openvpn3Session{
		ovpn3: openvpn3.NewSession(&openvpn3Callbacks{
			stateCallback: stateHandler,
			statsHandler:  statsHandler,
		}),
		config:        config,
		credsProvider: credentialsProvider,
		stateCallback: stateHandler,
	}
}

type openvpn3Session struct {
	ovpn3         *openvpn3.Session
	config        *ClientConfig
	credsProvider CredentialsProvider
	stateCallback Callback
}

func (session *openvpn3Session) Start() error {
	profile, err := session.config.ToConfigFileContent()
	if err != nil {
		return err
	}
	log.Info(openvpn3SessionPrefx, "Using client profile")
	log.Info(openvpn3SessionPrefx, profile)
	credentials := openvpn3.Credentials{}
	credentials.Username, credentials.Password, err = session.credsProvider()
	if err != nil {
		return err
	}

	session.ovpn3.Start(profile, credentials)
	return nil
}

func (session *openvpn3Session) Wait() error {
	defer session.stateCallback(ProcessExited)
	return session.ovpn3.Wait()
}

func (session *openvpn3Session) Stop() {
	session.ovpn3.Stop()
}

var _ Process = &openvpn3Session{}

type openvpn3Callbacks struct {
	stateCallback Callback
	statsHandler  SessionStatsHandler
}

func (callbacks *openvpn3Callbacks) Log(text string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		log.Info(openvpn3SessionPrefx, line)
	}
}

func (callbacks *openvpn3Callbacks) OnEvent(event openvpn3.Event) {
	log.Infof("%s%+v\n", openvpn3SessionPrefx, event)
	callbacks.stateCallback(State(event.Name))
}

func (callbacks *openvpn3Callbacks) OnStats(statistics openvpn3.Statistics) {
	stats := SessionStats{
		BytesSent:     statistics.BytesOut,
		BytesReceived: statistics.BytesIn,
	}
	callbacks.statsHandler(stats)
}

var _ openvpn3.Logger = &openvpn3Callbacks{}

var _ openvpn3.EventConsumer = &openvpn3Callbacks{}

var _ openvpn3.StatsConsumer = &openvpn3Callbacks{}
