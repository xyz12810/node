// +build !darwin

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
