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
package ship_server

const ShipgateHeaderSize = 8

type ShipgateHeader struct {
	Size uint16
	Type uint16
	Id   int
}

// Initial auth request sent to the shipgate.
type ShipgateAuthPkt struct {
	header ShipgateHeader
	name   []byte
}

// Contains the symmetric key from the shipgate.
type ShipgateKeyPkt struct {
	header ShipgateHeader
}

// Acknowldge that we got the key.
type ShipgateKeyAckPkt struct {
	header ShipgateHeader
}
