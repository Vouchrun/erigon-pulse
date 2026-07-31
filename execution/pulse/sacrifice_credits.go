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

package pulse

import (
	_ "embed"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/common/log/v3"
	"github.com/erigontech/erigon/execution/chain"
	"github.com/erigontech/erigon/execution/state"
	"github.com/erigontech/erigon/execution/tracing"
	"github.com/erigontech/erigon/execution/types/accounts"
)

// see https://gitlab.com/pulsechaincom/compressed-allocations/-/tags/Mainnet
//
//go:embed sacrifice_credits_mainnet.bin
var mainnetRawCredits []byte

// see https://gitlab.com/pulsechaincom/compressed-allocations/-/tags/Testnet-V4
//
//go:embed sacrifice_credits_testnet_v4.bin
var testnetV4RawCredits []byte

// applySacrificeCredits applies the sacrifice credits for the PrimordialPulse fork.
func applySacrificeCredits(state *state.IntraBlockState, pulseChainConfig *chain.PulseChainConfig, chainID *uint256.Int) error {
	rawCredits := mainnetRawCredits
	if chainID.Cmp(TestnetV4ChainID) == 0 {
		rawCredits = testnetV4RawCredits
	}

	if pulseChainConfig != nil && pulseChainConfig.Treasury != nil {
		balance, err := uint256.FromHex(pulseChainConfig.Treasury.Balance)
		if err != nil {
			return fmt.Errorf("pulsechain treasury balance: %w", err)
		}
		addr := accounts.InternAddress(common.HexToAddress(pulseChainConfig.Treasury.Addr))
		log.Info("Applying PrimordialPulse treasury allocation 💸", "addr", addr, "amount", balance)
		if err := state.AddBalance(addr, *balance, tracing.BalanceIncreaseSacrificeCredit); err != nil {
			return err
		}
	}

	log.Info("Applying PrimordialPulse sacrifice credits 💸")
	for ptr := 0; ptr < len(rawCredits); {
		byteCount := int(rawCredits[ptr])
		ptr++

		record := rawCredits[ptr : ptr+byteCount]
		ptr += byteCount

		addr := accounts.InternAddress(common.BytesToAddress(record[:20]))
		credit := new(uint256.Int).SetBytes(record[20:])
		if err := state.AddBalance(addr, *credit, tracing.BalanceIncreaseSacrificeCredit); err != nil {
			return err
		}
	}

	log.Info("Finished applying PrimordialPulse sacrifice credits 🤑")
	return nil
}
