// Copyright 2026 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package chain

import (
	"github.com/holiman/uint256"
)

// PulseChainConfig holds PulseChain-specific chain options (nil for all other chains).
type PulseChainConfig struct {
	// An optional treasury which will receive allocations during the PrimordialPulseBlock.
	Treasury *PulseChainTreasury `json:"treasury,omitempty"`
}

// PulseChainTreasury represents an optional treasury for launching PulseChain testnets.
type PulseChainTreasury struct {
	Addr    string `json:"addr"`
	Balance string `json:"balance"`
}

// PulseChainTTDOffset is a trivially small amount of work added to the Ethereum Mainnet TTD
// to allow for un-merging and merging with the PulseChain beacon chain.
var PulseChainTTDOffset = uint256.NewInt(131_072)
