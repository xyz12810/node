/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package openvpn

import (
	"github.com/MysteriumNetwork/openvpn3-go-bindings/openvpn3"
	log "github.com/cihub/seelog"
	"strings"
)

const openvpn3SessionPrefx = "[openvpn3 session] "

// NewClient creates openvpn client with given config params
func NewClient(openvpnBinary string, config *ClientConfig, stateHandler StateCallback, statsHandler SessionStatsHandler, credentialsProvider CredentialsProvider) Process {
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
	stateCallback StateCallback
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
	stateCallback StateCallback
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
