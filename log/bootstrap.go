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

package log

import "github.com/cihub/seelog"

const seewayLogXmlConfig = `
<seelog>
	<outputs formatid="main">
		<console/>
	</outputs>
	<formats>
		<format id="main" format="%UTCDate(2006-01-02T15:04:05.999999999) [%Level] %Msg%n"/>
	</formats>
</seelog>
`

// Bootstrap loads log package into the overall system
func Bootstrap() {
	var err error

	loggerRoot, err = seelog.LoggerFromConfigAsString(seewayLogXmlConfig)
	if err != nil {
		panic("Error parsing seelog configuration. " + err.Error())
	}

	err = seelog.UseLogger(loggerRoot)
	if err != nil {
		panic("Error setting new logger for seelog" + err.Error())
	}
}
