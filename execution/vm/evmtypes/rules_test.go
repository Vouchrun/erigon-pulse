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

package evmtypes

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/execution/chain"
)

func TestRulesPrimordialPulse(t *testing.T) {
	t.Parallel()

	cfg := &chain.Config{
		ChainID:              uint256.NewInt(369),
		ShanghaiTime:         common.NewUint64(1_683_786_515),
		PrimordialPulseBlock: common.NewUint64(17_233_000),
	}

	preFork := (&BlockContext{BlockNumber: 17_232_999, Time: 1_683_985_199}).Rules(cfg)
	assert.Equal(t, uint256.NewInt(1), preFork.ChainID, "pre-fork EVM chainID")
	assert.True(t, preFork.IsShanghai, "pre-fork block at post-mainnet-Shanghai time")

	preForkEarly := (&BlockContext{BlockNumber: 17_034_869, Time: 1_681_338_454}).Rules(cfg)
	assert.False(t, preForkEarly.IsShanghai, "pre-fork block before mainnet Shanghai time")

	forkBlock := (&BlockContext{BlockNumber: 17_233_000, Time: 1_683_985_200}).Rules(cfg)
	assert.Equal(t, uint256.NewInt(369), forkBlock.ChainID, "fork-block EVM chainID")
	assert.True(t, forkBlock.IsShanghai)

	postForkPreShanghai := (&BlockContext{BlockNumber: 17_233_000, Time: 1_683_786_514}).Rules(cfg)
	assert.False(t, postForkPreShanghai.IsShanghai, "at/after fork the chain shanghaiTime governs")
}
