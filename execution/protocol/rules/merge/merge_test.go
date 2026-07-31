// Copyright 2024 The Erigon Authors
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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package merge

import (
	"errors"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/common/log/v3"
	"github.com/erigontech/erigon/db/consensuschain"
	"github.com/erigontech/erigon/execution/chain"
	chainspec "github.com/erigontech/erigon/execution/chain/spec"
	"github.com/erigontech/erigon/execution/protocol/rules"
	"github.com/erigontech/erigon/execution/protocol/rules/ethash"
	"github.com/erigontech/erigon/execution/state"
	"github.com/erigontech/erigon/execution/tracing"
	"github.com/erigontech/erigon/execution/types"
	"github.com/erigontech/erigon/execution/types/accounts"
)

type readerMock struct{}

func (r readerMock) Config() *chain.Config {
	return nil
}

func (r readerMock) CurrentHeader() *types.Header {
	return nil
}

func (cr readerMock) CurrentFinalizedHeader() *types.Header {
	return nil
}

func (cr readerMock) CurrentSafeHeader() *types.Header {
	return nil
}

func (r readerMock) GetHeader(common.Hash, uint64) *types.Header {
	return nil
}

func (r readerMock) GetHeaderByNumber(uint64) *types.Header {
	return nil
}

func (r readerMock) GetHeaderByHash(common.Hash) *types.Header {
	return nil
}

func (r readerMock) GetTd(common.Hash, uint64) *uint256.Int {
	return nil
}

func (r readerMock) FrozenBlocks() uint64 {
	return 0
}
func (r readerMock) FrozenBorBlocks(align bool) uint64 { return 0 }

// The thing only that changes between normal ethash checks other than POW, is difficulty
// and nonce so we are gonna test those
func TestVerifyHeaderDifficulty(t *testing.T) {
	header := &types.Header{
		Difficulty: *common.Num1,
		Time:       1,
	}

	parent := &types.Header{}

	var eth1Engine rules.Engine
	mergeEngine := New(eth1Engine)

	err := mergeEngine.verifyHeader(readerMock{}, header, parent)
	if err != errInvalidDifficulty {
		if err != nil {
			t.Fatalf("Merge engine should not accept non-zero difficulty, got %s", err.Error())
		} else {
			t.Fatalf("Merge engine should not accept non-zero difficulty")
		}
	}
}

func TestVerifyHeaderNonce(t *testing.T) {
	header := &types.Header{
		Nonce:      types.BlockNonce{1, 0, 0, 0, 0, 0, 0, 0},
		Difficulty: *common.Num0,
		Time:       1,
	}

	parent := &types.Header{}

	var eth1Engine rules.Engine
	mergeEngine := New(eth1Engine)

	err := mergeEngine.verifyHeader(readerMock{}, header, parent)
	if err != errInvalidNonce {
		if err != nil {
			t.Fatalf("Merge engine should not accept non-zero difficulty, got %s", err.Error())
		} else {
			t.Fatalf("Merge engine should not accept non-zero difficulty")
		}
	}
}

func TestNullParentBeaconBlockRootDoesNotPanic(t *testing.T) {
	chainConfig := chainspec.Mainnet.Config
	header := &types.Header{ // fake PoS header *after* Cancun fork
		Difficulty: *ProofOfStakeDifficulty,
		Time:       *chainConfig.CancunTime + 1,
	}
	logger := log.New()
	chainReader := consensuschain.NewReader(chainConfig, nil, nil, logger) // tx and blockReader don't care
	systemCallCustom := func(contract accounts.Address, data []byte, ibs *state.IntraBlockState, header *types.Header, constCall bool) ([]byte, error) {
		return nil, nil
	}
	var intraBlockState state.IntraBlockState // don't care
	var tracer tracing.Hooks                  // don't care
	var eth1Engine rules.Engine
	mergeEngine := New(eth1Engine)
	err := mergeEngine.Initialize(chainConfig, chainReader, header, &intraBlockState, systemCallCustom, logger, &tracer)
	assert.NoError(t, err)
}

type pulseReaderMock struct{ readerMock }

func (r pulseReaderMock) Config() *chain.Config { return chainspec.Pulsechain.Config }

func (r pulseReaderMock) GetTd(common.Hash, uint64) *uint256.Int {
	return uint256.NewInt(1) // far below the PulseChain TTD
}

type mainnetReaderMock struct{ readerMock }

func (r mainnetReaderMock) Config() *chain.Config { return chainspec.Mainnet.Config }

func (r mainnetReaderMock) GetTd(common.Hash, uint64) *uint256.Int {
	return uint256.NewInt(1) // far below the mainnet TTD
}

func TestVerifyHeaderRoutesPoSBeforePulseTTD(t *testing.T) {
	// FullFake eth1 accepts anything, so nil means the eth1 route and
	// ErrUnknownAncestor (missing parent) means the PoS route.
	mergeEngine := New(ethash.NewFullFaker())

	posHeader := &types.Header{
		Number:     *uint256.NewInt(17_232_999),
		Difficulty: *common.Num0,
		Time:       1_683_985_199,
		ParentHash: common.HexToHash("0xdead"),
	}
	err := mergeEngine.VerifyHeader(pulseReaderMock{}, posHeader, false)
	if !errors.Is(err, rules.ErrUnknownAncestor) {
		t.Fatalf("pulse chain PoS header before TTD must take the PoS path (missing parent), got %v", err)
	}

	powHeader := &types.Header{
		Number:     *uint256.NewInt(17_233_000),
		Difficulty: *uint256.NewInt(131_072),
		Time:       1_683_985_200,
		ParentHash: common.HexToHash("0xbeef"),
	}
	if err := mergeEngine.VerifyHeader(pulseReaderMock{}, powHeader, false); err != nil {
		t.Fatalf("primordial header must route to the eth1 engine, got %v", err)
	}

	if err := mergeEngine.VerifyHeader(mainnetReaderMock{}, posHeader, false); err != nil {
		t.Fatalf("non-pulse chain PoS header before TTD must keep the upstream eth1 route, got %v", err)
	}
}
