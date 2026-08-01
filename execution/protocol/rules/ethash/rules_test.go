// Copyright 2017 The go-ethereum Authors
// (original work)
// Copyright 2024 The Erigon Authors
// (modifications)
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

package ethash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/holiman/uint256"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/common/empty"
	"github.com/erigontech/erigon/common/log/v3"
	"github.com/erigontech/erigon/common/math"
	"github.com/erigontech/erigon/execution/chain"
	"github.com/erigontech/erigon/execution/state"
	"github.com/erigontech/erigon/execution/types"
	"github.com/erigontech/erigon/execution/types/accounts"
)

type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   uint256.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber uint256.Int
	CurrentDifficulty  uint256.Int
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string
		ParentDifficulty   string
		CurrentTimestamp   string
		CurrentBlocknumber string
		CurrentDifficulty  string
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = math.MustParseUint64(ext.ParentTimestamp)
	d.ParentDifficulty = *uint256.MustFromHex(ext.ParentDifficulty)
	d.CurrentTimestamp = math.MustParseUint64(ext.CurrentTimestamp)
	d.CurrentBlocknumber = *uint256.MustFromHex(ext.CurrentBlocknumber)
	d.CurrentDifficulty = *uint256.MustFromHex(ext.CurrentDifficulty)

	return nil
}

func TestCalcDifficulty(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "tests", "testdata", "BasicTests", "difficulty.json"))
	if err != nil {
		t.Skip(err)
	}
	defer file.Close()

	tests := make(map[string]diffTest)
	err = json.NewDecoder(file).Decode(&tests)
	if err != nil {
		t.Fatal(err)
	}

	homestead := uint64(1150000)
	config := &chain.Config{HomesteadBlock: &homestead}

	for name, test := range tests {
		number := test.CurrentBlocknumber.Uint64() - 1
		diff := CalcDifficulty(config, test.CurrentTimestamp,
			test.ParentTimestamp,
			test.ParentDifficulty,
			number,
			empty.UncleHash,
		)
		if diff.Cmp(&test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", &test.CurrentDifficulty, "and calculated", &diff)
		}
	}
}

func TestCalcDifficultyPrimordialPulse(t *testing.T) {
	t.Parallel()
	cfg := &chain.Config{
		HomesteadBlock:       common.NewUint64(1_150_000),
		GrayGlacierBlock:     common.NewUint64(15_050_000),
		PrimordialPulseBlock: common.NewUint64(17_233_000),
	}
	d := CalcDifficulty(cfg, 1_683_985_200, 1_683_985_190, *uint256.NewInt(0), 17_232_999, empty.UncleHash)
	if d.Cmp(chain.PulseChainTTDOffset) != 0 {
		t.Errorf("primordial block difficulty: got %v, want %v", &d, chain.PulseChainTTDOffset)
	}

	cfgNoPulse := &chain.Config{
		HomesteadBlock:   common.NewUint64(1_150_000),
		GrayGlacierBlock: common.NewUint64(15_050_000),
	}
	d2 := CalcDifficulty(cfgNoPulse, 1_683_985_200, 1_683_985_190, *uint256.NewInt(0), 17_232_999, empty.UncleHash)
	if d2.Cmp(chain.PulseChainTTDOffset) == 0 {
		t.Error("without PrimordialPulseBlock the difficulty must not equal the pulse offset constant")
	}
}

func TestFinalizeAppliesPrimordialPulse(t *testing.T) {
	t.Parallel()
	cfg := &chain.Config{
		ChainID:              uint256.NewInt(369),
		ByzantiumBlock:       common.NewUint64(4_370_000),
		ConstantinopleBlock:  common.NewUint64(7_280_000),
		PrimordialPulseBlock: common.NewUint64(17_233_000),
		PulseChain:           &chain.PulseChainConfig{},
	}
	engine := &Ethash{}
	depositContract := accounts.InternAddress(common.HexToAddress("0x3693693693693693693693693693693693693693"))

	ibs := state.New(state.NewNoopReader())
	header := &types.Header{Number: *uint256.NewInt(17_233_000), Time: 1_683_985_200}
	if _, err := engine.Finalize(cfg, header, ibs, nil, nil, nil, nil, nil, false, log.New()); err != nil {
		t.Fatal(err)
	}
	size, err := ibs.GetCodeSize(depositContract)
	if err != nil {
		t.Fatal(err)
	}
	if size == 0 {
		t.Error("deposit contract not deployed after Finalize at the primordial block")
	}

	ibs2 := state.New(state.NewNoopReader())
	header2 := &types.Header{Number: *uint256.NewInt(17_233_001), Time: 1_683_985_210}
	if _, err := engine.Finalize(cfg, header2, ibs2, nil, nil, nil, nil, nil, false, log.New()); err != nil {
		t.Fatal(err)
	}
	exist, err := ibs2.Exist(depositContract)
	if err != nil {
		t.Fatal(err)
	}
	if exist {
		t.Error("the transition must not apply past the primordial block")
	}
}
