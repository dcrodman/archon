/*
* Archon Ship Server
* Copyright (C) 2014 Andrew Rodman
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
* ---------------------------------------------------------------------
 */
package ship

// Packet types for the shipgate. These can overlap since they aren't
// processed by the same set of handlers as the client ones.
const (
	ShipgateHeaderSize = 8
	ShipgateAuthType   = 0x01
	ShipgateAuthAck    = 0x02
)

type ShipgateHeader struct {
	Size uint16
	Type uint16
	Id   uint32
}

// Initial auth request sent to the shipgate.
type ShipgateAuthPkt struct {
	Header ShipgateHeader
	Name   [24]byte
}
