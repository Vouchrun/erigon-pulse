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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erigontech/erigon/common"
)

func TestPrimordialPulseBlockChecks(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	assert.False(t, cfg.IsPrimordialPulseBlock(0), "nil PrimordialPulseBlock")
	assert.False(t, cfg.PrimordialPulseAhead(0), "nil PrimordialPulseBlock")

	cfg.PrimordialPulseBlock = common.NewUint64(17_233_000)

	assert.False(t, cfg.IsPrimordialPulseBlock(17_232_999))
	assert.True(t, cfg.IsPrimordialPulseBlock(17_233_000))
	assert.False(t, cfg.IsPrimordialPulseBlock(17_233_001))

	assert.True(t, cfg.PrimordialPulseAhead(0))
	assert.True(t, cfg.PrimordialPulseAhead(17_232_999))
	assert.False(t, cfg.PrimordialPulseAhead(17_233_000), "fork block itself is not ahead")
	assert.False(t, cfg.PrimordialPulseAhead(17_233_001))
}

func TestIsShanghaiAtPrimordialPulse(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ShanghaiTime:         common.NewUint64(1_683_786_515),
		PrimordialPulseBlock: common.NewUint64(17_233_000),
	}
	assert.True(t, cfg.IsShanghaiAt(17_232_999, 1_681_338_455), "pre-fork blocks use the Ethereum mainnet Shanghai time")
	assert.False(t, cfg.IsShanghaiAt(17_232_999, 1_681_338_454))
	assert.True(t, cfg.IsShanghaiAt(17_233_000, 1_683_786_515), "fork block uses the chain shanghaiTime")
	assert.False(t, cfg.IsShanghaiAt(17_233_000, 1_683_786_514))

	plain := &Config{ShanghaiTime: common.NewUint64(1_683_786_515)}
	assert.True(t, plain.IsShanghaiAt(1, 1_683_786_515), "no primordial config: plain IsShanghai behavior")
	assert.False(t, plain.IsShanghaiAt(1, 1_683_786_514))
}
