/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package noop

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/urfave/cli"
)

// Options describes options which are required to start Noop service
type Options struct{}

// ParseCLIFlags function fills in Noop options from CLI context
func ParseCLIFlags(_ *cli.Context) service.Options {
	return Options{}
}

// ParseJSONOptions function fills in Noop options from JSON request
func ParseJSONOptions(request json.RawMessage) (service.Options, error) {
	var opts Options
	err := json.Unmarshal(request, &opts)
	return opts, err
}
