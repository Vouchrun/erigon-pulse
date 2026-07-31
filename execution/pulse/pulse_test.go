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
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/execution/chain"
	"github.com/erigontech/erigon/execution/state"
	"github.com/erigontech/erigon/execution/types/accounts"
)

func TestReplaceDepositContract(t *testing.T) {
	t.Parallel()
	ibs := state.New(state.NewNoopReader())
	require.NoError(t, replaceDepositContract(ibs))

	code, err := ibs.GetCode(pulseDepositContractAddr)
	require.NoError(t, err)
	assert.Equal(t, depositContractBytes, code, "new deposit contract code")

	nonce, err := ibs.GetNonce(pulseDepositContractAddr)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), nonce)

	balance, err := ibs.GetBalance(pulseDepositContractAddr)
	require.NoError(t, err)
	assert.True(t, balance.IsZero(), "new deposit contract balance must be zero")

	for _, slot := range depositContractStorage {
		got, err := ibs.GetState(pulseDepositContractAddr, accounts.InternKey(common.HexToHash(slot[0])))
		require.NoError(t, err)
		want, err := uint256.FromHex(slot[1])
		require.NoError(t, err)
		assert.Equal(t, *want, got, "storage slot %s", slot[0])
	}

	stubCode, err := ibs.GetCode(ethereumDepositContractAddr)
	require.NoError(t, err)
	assert.Equal(t, nilContractBytes, stubCode, "old deposit contract must be replaced by the reverting stub")
}

func TestApplySacrificeCreditsMainnet(t *testing.T) {
	t.Parallel()
	ibs := state.New(state.NewNoopReader())
	require.NoError(t, applySacrificeCredits(ibs, &chain.PulseChainConfig{}, MainnetChainID))

	// sum credits per address in case the compressed list repeats an address
	type sumKey = common.Address
	sums := map[sumKey]*uint256.Int{}
	count := 0
	for ptr := 0; ptr < len(mainnetRawCredits); {
		byteCount := int(mainnetRawCredits[ptr])
		ptr++
		record := mainnetRawCredits[ptr : ptr+byteCount]
		ptr += byteCount

		addr := common.BytesToAddress(record[:20])
		credit := new(uint256.Int).SetBytes(record[20:])
		if cur, ok := sums[addr]; ok {
			cur.Add(cur, credit)
		} else {
			sums[addr] = credit
		}
		count++
	}

	assert.Greater(t, count, 100_000, "expected a large number of sacrifice records")
	for addr, want := range sums {
		balance, err := ibs.GetBalance(accounts.InternAddress(addr))
		require.NoError(t, err)
		assert.Equal(t, *want, balance, "credited balance mismatch for %x", addr)
	}
}

func TestApplySacrificeCreditsTestnetTreasury(t *testing.T) {
	t.Parallel()
	cfg := &chain.PulseChainConfig{
		Treasury: &chain.PulseChainTreasury{
			Addr:    "0xA592ED65885bcbCeb30442F4902a0D1Cf3AcB8fC",
			Balance: "0x314DC6448D9338C15B0A00000000",
		},
	}
	ibs := state.New(state.NewNoopReader())
	require.NoError(t, applySacrificeCredits(ibs, cfg, TestnetV4ChainID))

	treasuryAddr := accounts.InternAddress(common.HexToAddress(cfg.Treasury.Addr))
	want, err := uint256.FromHex(cfg.Treasury.Balance)
	require.NoError(t, err)
	balance, err := ibs.GetBalance(treasuryAddr)
	require.NoError(t, err)
	assert.Equal(t, *want, balance, "treasury balance")

	// spot-check the first and last testnet records
	for _, ptr := range firstAndLastRecordOffsets(testnetV4RawCredits) {
		byteCount := int(testnetV4RawCredits[ptr])
		record := testnetV4RawCredits[ptr+1 : ptr+1+byteCount]
		addr := accounts.InternAddress(common.BytesToAddress(record[:20]))
		credit := new(uint256.Int).SetBytes(record[20:])
		balance, err := ibs.GetBalance(addr)
		require.NoError(t, err)
		assert.Equal(t, *credit, balance, "record at offset %d", ptr)
	}
}

func firstAndLastRecordOffsets(raw []byte) []int {
	first := 0
	last := 0
	for ptr := 0; ptr < len(raw); {
		byteCount := int(raw[ptr])
		last = ptr
		ptr += 1 + byteCount
	}
	return []int{first, last}
}
