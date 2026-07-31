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

// Package pulse implements the PulseChain fork.
package pulse

import (
	"github.com/holiman/uint256"

	"github.com/erigontech/erigon/execution/chain"
	"github.com/erigontech/erigon/execution/state"
)

var (
	MainnetChainID   = uint256.NewInt(369)
	TestnetV4ChainID = uint256.NewInt(943)
)

// PrimordialPulseFork applies the PrimordialPulse fork changes.
func PrimordialPulseFork(state *state.IntraBlockState, pulseChainConfig *chain.PulseChainConfig, chainID *uint256.Int) error {
	if err := applySacrificeCredits(state, pulseChainConfig, chainID); err != nil {
		return err
	}
	return replaceDepositContract(state)
}
