// Copyright 2015 Dorival Pedroso and Raul Durand. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mreten

import (
	"testing"

	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/plt"
)

func Test_vg01(tst *testing.T) {

	//verbose()
	chk.PrintTitle("vg01")

	mdl := GetModel("testsim", "mat1", "vg", false)
	prm := mdl.GetPrms(true)
	slmax := prm.Find("slmax")
	slmax.V = 0.95
	err := mdl.Init(prm)
	if err != nil {
		tst.Errorf("test failed: %v\n", err)
		return
	}

	ref := GetModel("testsim", "mat1", "ref-m1", false)
	err = ref.Init(ref.GetPrms(true))
	if err != nil {
		tst.Errorf("test failed: %v\n", err)
		return
	}

	pc0 := -5.0
	sl0 := mdl.SlMax()
	pcf := 20.0
	nptsA := 41
	nptsB := 11

	if chk.Verbose {
		plt.Reset()
		Plot(ref, pc0, sl0, pcf, nptsA, "'k--'", "'k--'", "ref-m1")
		Plot(mdl, pc0, sl0, pcf, nptsA, "'b.-'", "'r+-'", "vg")
	}

	tolCc := 1e-10
	tolD1a, tolD1b := 1e-10, 1e-17
	tolD2a, tolD2b := 1e-10, 1e-17
	Check(tst, mdl, pc0, sl0, pcf, nptsB, tolCc, tolD1a, tolD1b, tolD2a, tolD2b, chk.Verbose, []float64{}, 1e-7, chk.Verbose)

	if chk.Verbose {
		PlotEnd(true)
	}
}
