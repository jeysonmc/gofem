// Copyright 2015 Dorival Pedroso and Raul Durand. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fem

import (
	"math"

	"github.com/cpmech/gofem/inp"

	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/fun"
	"github.com/cpmech/gosl/la"
)

// Beam represents a structural beam element (Euler-Bernoulli, linear elastic)
type Beam struct {

	// basic data
	Cell *inp.Cell   // the cell structure
	X    [][]float64 // matrix of nodal coordinates [ndim][nnode]
	Nu   int         // total number of unknowns == 2 * nsn
	Ndim int         // space dimension

	// parameters
	E   float64 // Young's modulus
	A   float64 // cross-sectional area
	Izz float64 // Inertia zz

	// variables for dynamics
	Rho  float64  // density of solids
	Gfcn fun.Func // gravity function

	// vectors and matrices
	T   [][]float64 // global-to-local transformation matrix [nnode*ndim][nnode*ndim]
	Kl  [][]float64 // local K matrix
	K   [][]float64 // global K matrix
	Ml  [][]float64 // local M matrices
	M   [][]float64 // global M matrices
	Rus []float64   // residual: Rus = fi - fx

	// problem variables
	Umap []int    // assembly map (location array/element equations)
	Hasq bool     // has distributed loads
	QnL  fun.Func // distributed normal load functions: left
	QnR  fun.Func // distributed normal load functions: right
	Qt   fun.Func // distributed tangential load

	// scratchpad. computed @ each ip
	grav []float64 // [ndim] gravity vector
	fi   []float64 // [nu] internal forces
	ue   []float64 // local u vector
	ζe   []float64 // local ζ* vector
	fxl  []float64 // local external force vector
}

// register element
func init() {

	// information allocator
	infogetters["beam"] = func(sim *inp.Simulation, cell *inp.Cell, edat *inp.ElemData) *Info {

		// new info
		var info Info

		// solution variables
		ykeys := []string{"ux", "uy", "rz"}
		if sim.Ndim == 3 {
			ykeys = []string{"ux", "uy", "uz", "rx", "ry", "rz"}
		}
		info.Dofs = make([][]string, 2)
		for m := 0; m < 2; m++ {
			info.Dofs[m] = ykeys
		}

		// maps
		info.Y2F = map[string]string{"ux": "fx", "uy": "fy", "uz": "fz", "rx": "mx", "ry": "my", "rz": "mz"}

		// t1 and t2 variables
		info.T2vars = ykeys
		return &info
	}

	// element allocator
	eallocators["beam"] = func(sim *inp.Simulation, cell *inp.Cell, edat *inp.ElemData, x [][]float64) Elem {

		// check
		ndim := len(x)
		if ndim == 3 {
			chk.Panic("beam is not implemented for 3D yet")
		}

		// basic data
		var o Beam
		o.Cell = cell
		o.X = x
		ndof := 3 * (ndim - 1)
		o.Nu = ndof * ndim
		o.Ndim = ndim

		// parameters
		matdata := sim.MatParams.Get(edat.Mat)
		if matdata == nil {
			return nil
		}
		for _, p := range matdata.Prms {
			switch p.N {
			case "E":
				o.E = p.V
			case "A":
				o.A = p.V
			case "Izz":
				o.Izz = p.V
			case "rho":
				o.Rho = p.V
			}
		}

		// vectors and matrices
		o.T = la.MatAlloc(o.Nu, o.Nu)
		o.Kl = la.MatAlloc(o.Nu, o.Nu)
		o.K = la.MatAlloc(o.Nu, o.Nu)
		o.Ml = la.MatAlloc(o.Nu, o.Nu)
		o.M = la.MatAlloc(o.Nu, o.Nu)
		o.ue = make([]float64, o.Nu)
		o.ζe = make([]float64, o.Nu)
		o.fxl = make([]float64, o.Nu)
		o.Rus = make([]float64, o.Nu)

		// T
		dx := o.X[0][1] - o.X[0][0]
		dy := o.X[1][1] - o.X[1][0]
		l := math.Sqrt(dx*dx + dy*dy)
		c := dx / l
		s := dy / l
		o.T[0][0] = c
		o.T[0][1] = s
		o.T[1][0] = -s
		o.T[1][1] = c
		o.T[2][2] = 1
		o.T[3][3] = c
		o.T[3][4] = s
		o.T[4][3] = -s
		o.T[4][4] = c
		o.T[5][5] = 1

		// aux vars
		ll := l * l
		m := o.E * o.A / l
		n := o.E * o.Izz / (ll * l)

		// K
		o.Kl[0][0] = m
		o.Kl[0][3] = -m
		o.Kl[1][1] = 12 * n
		o.Kl[1][2] = 6 * l * n
		o.Kl[1][4] = -12 * n
		o.Kl[1][5] = 6 * l * n
		o.Kl[2][1] = 6 * l * n
		o.Kl[2][2] = 4 * ll * n
		o.Kl[2][4] = -6 * l * n
		o.Kl[2][5] = 2 * ll * n
		o.Kl[3][0] = -m
		o.Kl[3][3] = m
		o.Kl[4][1] = -12 * n
		o.Kl[4][2] = -6 * l * n
		o.Kl[4][4] = 12 * n
		o.Kl[4][5] = -6 * l * n
		o.Kl[5][1] = 6 * l * n
		o.Kl[5][2] = 2 * ll * n
		o.Kl[5][4] = -6 * l * n
		o.Kl[5][5] = 4 * ll * n
		la.MatTrMul3(o.K, 1, o.T, o.Kl, o.T) // K := 1 * trans(T) * Kl * T

		// M
		m = o.Rho * o.A * l / 420.0
		o.Ml[0][0] = 140.0 * m
		o.Ml[0][3] = 70.0 * m
		o.Ml[1][1] = 156.0 * m
		o.Ml[1][2] = 22.0 * l * m
		o.Ml[1][4] = 54.0 * m
		o.Ml[1][5] = -13.0 * l * m
		o.Ml[2][1] = 22.0 * l * m
		o.Ml[2][2] = 4.0 * ll * m
		o.Ml[2][4] = 13.0 * l * m
		o.Ml[2][5] = -3.0 * ll * m
		o.Ml[3][0] = 70.0 * m
		o.Ml[3][3] = 140.0 * m
		o.Ml[4][1] = 54.0 * m
		o.Ml[4][2] = 13.0 * l * m
		o.Ml[4][4] = 156.0 * m
		o.Ml[4][5] = -22.0 * l * m
		o.Ml[5][1] = -13.0 * l * m
		o.Ml[5][2] = -3.0 * ll * m
		o.Ml[5][4] = -22.0 * l * m
		o.Ml[5][5] = 4.0 * ll * m
		la.MatTrMul3(o.M, 1, o.T, o.Ml, o.T) // M := 1 * trans(T) * Ml * T

		// scratchpad. computed @ each ip
		o.grav = make([]float64, ndim)
		o.fi = make([]float64, o.Nu)

		// return new element
		return &o
	}
}

// Id returns the cell Id
func (o Beam) Id() int { return o.Cell.Id }

// SetEqs set equations [2][?]. Format of eqs == format of info.Dofs
func (o *Beam) SetEqs(eqs [][]int, mixedform_eqs []int) (err error) {
	ndof := 3 * (o.Ndim - 1)
	o.Umap = make([]int, o.Nu)
	for m := 0; m < 2; m++ {
		for i := 0; i < ndof; i++ {
			r := i + m*ndof
			o.Umap[r] = eqs[m][i]
		}
	}
	return
}

// SetEleConds set element conditions
func (o *Beam) SetEleConds(key string, f fun.Func, extra string) (err error) {

	// gravity
	if key == "g" {
		o.Gfcn = f
		return
	}

	// distributed loads
	switch key {
	case "qn":
		o.Hasq, o.QnL, o.QnR = true, f, f
	case "qnL":
		o.Hasq, o.QnL = true, f
	case "qnR":
		o.Hasq, o.QnR = true, f
	case "qt":
		o.Hasq, o.Qt = true, f
	default:
		return chk.Err("cannot handle boundary condition named %q", key)
	}
	return
}

// InterpStarVars interpolates star variables to integration points
func (o *Beam) InterpStarVars(sol *Solution) (err error) {
	for i, I := range o.Umap {
		o.ζe[i] = sol.Zet[I]
	}
	return
}

// adds -R to global residual vector fb
func (o Beam) AddToRhs(fb []float64, sol *Solution) (err error) {

	// node displacements
	for i, I := range o.Umap {
		o.ue[i] = sol.Y[I]
	}

	// steady/dynamics
	if sol.Steady {
		la.MatVecMul(o.fi, 1, o.K, o.ue)
	} else {
		α1 := sol.DynCfs.α1
		for i := 0; i < o.Nu; i++ {
			o.fi[i] = 0
			for j := 0; j < o.Nu; j++ {
				o.fi[i] += o.M[i][j]*(α1*o.ue[j]-o.ζe[j]) + o.K[i][j]*o.ue[j]
			}
		}
	}

	// distributed loads
	if o.Hasq {
		dx := o.X[0][1] - o.X[0][0]
		dy := o.X[1][1] - o.X[1][0]
		l := math.Sqrt(dx*dx + dy*dy)
		qnL := o.QnL.F(sol.T, nil)
		qnR := o.QnR.F(sol.T, nil)
		qt := o.Qt.F(sol.T, nil)
		o.fxl[0] = qt * l / 2.0
		o.fxl[1] = l * (7.0*qnL + 3.0*qnR) / 20.0
		o.fxl[2] = l * l * (3.0*qnL + 2.0*qnR) / 60.0
		o.fxl[3] = qt * l / 2.0
		o.fxl[4] = l * (3.0*qnL + 7.0*qnR) / 20.0
		o.fxl[5] = -l * l * (2.0*qnL + 3.0*qnR) / 60.0
		la.MatTrVecMulAdd(o.fi, -1.0, o.T, o.fxl) // Rus -= fx; fx = trans(T) * fxl
	}

	// add to fb
	for i, I := range o.Umap {
		fb[I] -= o.fi[i]
	}
	return
}

// adds element K to global Jacobian matrix Kb
func (o Beam) AddToKb(Kb *la.Triplet, sol *Solution, firstIt bool) (err error) {
	if sol.Steady {
		for i, I := range o.Umap {
			for j, J := range o.Umap {
				Kb.Put(I, J, o.K[i][j])
			}
		}
		return
	}
	α1 := sol.DynCfs.α1
	for i, I := range o.Umap {
		for j, J := range o.Umap {
			Kb.Put(I, J, o.M[i][j]*α1+o.K[i][j])
		}
	}
	return
}

// Update perform (tangent) update
func (o *Beam) Update(sol *Solution) (err error) {
	return
}

// Encode encodes internal variables
func (o Beam) Encode(enc Encoder) (err error) {
	return
}

// Decode decodes internal variables
func (o Beam) Decode(dec Decoder) (err error) {
	return
}

// OutIpsData returns data from all integration points for output
func (o Beam) OutIpsData() (data []*OutIpData) {
	return
}
