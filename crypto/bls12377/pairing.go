package bls12377

type pair struct {
	g1 *PointG1
	g2 *PointG2
}

func newPair(g1 *PointG1, g2 *PointG2) pair {
	return pair{g1, g2}
}

// Engine is BLS12-381 elliptic curve pairing engine
type Engine struct {
	G1   *G1
	G2   *G2
	fp12 *fp12
	fp2  *fp2
	pairingEngineTemp
	pairs []pair
}

// NewPairingEngine creates new pairing engine instance.
func NewPairingEngine() *Engine {
	fp2 := newFp2()
	fp6 := newFp6(fp2)
	fp12 := newFp12(fp6)
	g1 := NewG1()
	g2 := newG2(fp2)
	return &Engine{
		fp2:               fp2,
		fp12:              fp12,
		G1:                g1,
		G2:                g2,
		pairingEngineTemp: newEngineTemp(),
	}
}

type pairingEngineTemp struct {
	t2  [10]*fe2
	t12 [9]fe12
}

func newEngineTemp() pairingEngineTemp {
	t2 := [10]*fe2{}
	for i := 0; i < 10; i++ {
		t2[i] = &fe2{}
	}
	t12 := [9]fe12{}
	return pairingEngineTemp{t2, t12}
}

// AddPair adds a g1, g2 point pair to pairing engine
func (e *Engine) AddPair(g1 *PointG1, g2 *PointG2) *Engine {
	p := newPair(g1, g2)
	if !e.isZero(p) {
		e.affine(p)
		e.pairs = append(e.pairs, p)
	}
	return e
}

// AddPairInv adds a G1, G2 point pair to pairing engine. G1 point is negated.
func (e *Engine) AddPairInv(g1 *PointG1, g2 *PointG2) *Engine {
	ng1 := e.G1.New().Set(g1)
	e.G1.Neg(ng1, g1)
	e.AddPair(ng1, g2)
	return e
}

// Reset deletes added pairs.
func (e *Engine) Reset() *Engine {
	e.pairs = []pair{}
	return e
}

func (e *Engine) isZero(p pair) bool {
	return e.G1.IsZero(p.g1) || e.G2.IsZero(p.g2)
}

func (e *Engine) affine(p pair) {
	e.G1.Affine(p.g1)
	e.G2.Affine(p.g2)
}

func (e *Engine) doublingStep(coeff *fe6, r *PointG2) {

	fp2 := e.fp2
	t := e.t2
	fp2.mul(t[0], &r[0], &r[1])
	fp2.mul0(t[0], t[0], twoInv)
	fp2.square(t[1], &r[1])
	fp2.square(t[2], &r[2])
	fp2.double(t[7], t[2])
	fp2.add(t[7], t[7], t[2])
	fp2.mul(t[3], t[7], b2)
	fp2.double(t[4], t[3])
	fp2.add(t[4], t[4], t[3])
	fp2.add(t[5], t[1], t[4])
	fp2.mul0(t[5], t[5], twoInv)
	fp2.add(t[6], &r[1], &r[2])
	fp2.square(t[6], t[6])
	fp2.add(t[7], t[2], t[1])
	fp2.sub(t[6], t[6], t[7])
	fp2.sub(&coeff[2], t[3], t[1])
	fp2.square(t[7], &r[0])
	fp2.sub(t[4], t[1], t[4])
	fp2.mul(&r[0], t[4], t[0])
	fp2.square(t[2], t[3])
	fp2.double(t[3], t[2])
	fp2.add(t[3], t[3], t[2])
	fp2.square(t[5], t[5])
	fp2.sub(&r[1], t[5], t[3])
	fp2.mul(&r[2], t[1], t[6])
	fp2.double(t[0], t[7])
	fp2.add(&coeff[1], t[0], t[7])
	fp2.neg(&coeff[0], t[6])
}

func (e *Engine) additionStep(coeff *fe6, r, q *PointG2) {

	fp2 := e.fp2
	t := e.t2
	fp2.mul(t[0], &q[1], &r[2])
	fp2.neg(t[0], t[0])
	fp2.add(t[0], t[0], &r[1])
	fp2.mul(t[1], &q[0], &r[2])
	fp2.neg(t[1], t[1])
	fp2.add(t[1], t[1], &r[0])
	fp2.square(t[2], t[0])
	fp2.square(t[3], t[1])
	fp2.mul(t[4], t[1], t[3])
	fp2.mul(t[2], &r[2], t[2])
	fp2.mul(t[3], &r[0], t[3])
	fp2.double(t[5], t[3])
	fp2.sub(t[5], t[4], t[5])
	fp2.add(t[5], t[5], t[2])
	fp2.mul(&r[0], t[1], t[5])
	fp2.sub(t[2], t[3], t[5])
	fp2.mul(t[2], t[2], t[0])
	fp2.mul(t[3], &r[1], t[4])
	fp2.sub(&r[1], t[2], t[3])
	fp2.mul(&r[2], &r[2], t[4])
	fp2.mul(t[2], t[1], &q[1])
	fp2.mul(t[3], t[0], &q[0])
	fp2.sub(&coeff[2], t[3], t[2])
	fp2.neg(&coeff[1], t[0])
	coeff[0].set(t[1])
}

func (e *Engine) precompute() [][69]fe6 {
	n := len(e.pairs)
	coeffs := make([][69]fe6, len(e.pairs))
	for i := 0; i < n; i++ {
		r := new(PointG2).Set(e.pairs[i].g2)
		j := 0
		for k := 62; k >= 0; k-- {
			e.doublingStep(&coeffs[i][j], r)
			if x.Bit(k) != 0 {
				j++
				e.additionStep(&coeffs[i][j], r, e.pairs[i].g2)
			}
			j++
		}
	}
	return coeffs
}

func (e *Engine) lineEval(f *fe12, coeffs [][69]fe6, j int) {
	t := e.t2
	for i := 0; i < len(e.pairs); i++ {
		e.fp2.mul0(t[0], &coeffs[i][j][0], &e.pairs[i].g1[1])
		e.fp2.mul0(t[1], &coeffs[i][j][1], &e.pairs[i].g1[0])
		e.fp12.mul034(f, t[0], t[1], &coeffs[i][j][2])
	}
}

func (e *Engine) millerLoop(f *fe12) {
	coeffs := e.precompute()
	f.one()
	j := 0
	for i := 62; i >= 0; i-- {
		if i != 62 {
			e.fp12.square(f, f)
		}
		e.lineEval(f, coeffs, j)
		if x.Bit(i) != 0 {
			j++
			e.lineEval(f, coeffs, j)
		}
		j++
	}
}

// exp raises element by x = 9586122913090633729
func (e *Engine) exp(c, a *fe12) {
	fp12, t := e.fp12, e.fp12.t12
	fp12.cyclotomicSquare(t[0], a)    // t0 = 2
	fp12.mul(t[2], t[0], a)           // t2 = 3
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) // t0 = 32
	fp12.mul(t[3], t[0], a)           // t3 = 33
	fp12.cyclotomicSquare(t[0], t[3]) // t0 = 66
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) //
	fp12.cyclotomicSquare(t[0], t[0]) // t0 = 4224
	fp12.mul(t[0], t[0], t[3])        // t0 = 4257
	fp12.cyclotomicSquare(t[0], t[0])
	fp12.cyclotomicSquare(t[0], t[0])
	fp12.cyclotomicSquare(t[0], t[0])
	fp12.cyclotomicSquare(t[0], t[0])
	fp12.cyclotomicSquare(t[0], t[0]) // t0 = 136224
	fp12.mul(t[0], t[0], t[2])        // t0 = 136227
	for i := 0; i < 46; i++ {
		fp12.cyclotomicSquare(t[0], t[0]) // t0 = 136227 ** 46
	}
	fp12.mul(c, t[0], a) // out = 9586122913090633729
}

func (e *Engine) finalExp(f *fe12) {

	fp12, t := e.fp12, e.t12
	// easy part

	fp12.inverse(&t[1], f)        // t1 = f0 ^ -1
	fp12.conjugate(&t[0], f)      // t0 = f0 ^ p6
	fp12.mul(&t[2], &t[0], &t[1]) // t2 = f0 ^ (p6 - 1)
	t[1].set(&t[2])               // t1 = f0 ^ (p6 - 1)
	fp12.frobeniusMap2(&t[2])     // t2 = f0 ^ ((p6 - 1) * p2)
	fp12.mul(&t[2], &t[2], &t[1]) // t2 = f0 ^ ((p6 - 1) * (p2 + 1))

	// f = f0 ^ ((p6 - 1) * (p2 + 1))

	// hard part
	// https://eprint.iacr.org/2016/130
	// On the Computation of the Optimal Ate Pairing at the 192-bit Security Level
	// Section 3
	// f ^ d = ??_0 + ??_1 * p + ??_2 * p^2 + ??_3 * p^3

	fp12.conjugate(&t[1], &t[2])
	fp12.cyclotomicSquare(&t[1], &t[1]) // t1 = f ^ (-2)
	e.exp(&t[3], &t[2])                 // t3 = f ^ (u)
	fp12.cyclotomicSquare(&t[4], &t[3]) // t4 = f ^ (2u)
	fp12.mul(&t[5], &t[1], &t[3])       // t5 = f ^ (u - 2)
	e.exp(&t[1], &t[5])                 // t1 = f ^ (u^2 - 2 * u)
	e.exp(&t[0], &t[1])                 // t0 = f ^ (u^3 - 2 * u^2)
	e.exp(&t[6], &t[0])                 // t6 = f ^ (u^4 - 2 * u^3)
	fp12.mul(&t[6], &t[6], &t[4])       // t6 = f ^ (u^4 - 2 * u^3 + 2 * u)
	e.exp(&t[4], &t[6])                 // t4 = f ^ (u^4 - 2 * u^3 + 2 * u^2)
	fp12.conjugate(&t[5], &t[5])        // t5 = f ^ (2 - u)
	fp12.mul(&t[4], &t[4], &t[5])       // t4 = f ^ (u^4 - 2 * u^3 + 2 * u^2 - u + 2)
	fp12.mul(&t[4], &t[4], &t[2])       // f_??_0 = t4 = f ^ (u^4 - 2 * u^3 + 2 * u^2 - u + 3)

	fp12.conjugate(&t[5], &t[2])  // t5 = f ^ (-1)
	fp12.mul(&t[5], &t[5], &t[6]) // t1  = f ^ (u^4 - 2 * u^3 + 2 * u - 1)
	fp12.frobeniusMap1(&t[5])     // f_??_1 = t1 = f ^ ((u^4 - 2 * u^3 + 2 * u - 1) ^ p)

	fp12.mul(&t[3], &t[3], &t[0]) // t3 = f ^ (u^3 - 2 * u^2 + u)
	fp12.frobeniusMap2(&t[3])     // f_??_2 = t3 = f ^ ((u^3 - 2 * u^2 + u) ^ p^2)

	fp12.mul(&t[1], &t[1], &t[2]) // t1 = f ^ (u^2 - 2 * u + 1)
	fp12.frobeniusMap3(&t[1])     // f_??_3 = t1 = f ^ ((u^2 - 2 * u + 1) ^ p^3)

	// out = f ^ (??_0 + ??_1 + ??_2 + ??_3)
	fp12.mul(&t[3], &t[3], &t[1])
	fp12.mul(&t[3], &t[3], &t[5])
	fp12.mul(f, &t[3], &t[4])
}

func (e *Engine) calculate() *fe12 {
	f := e.fp12.one()
	if len(e.pairs) == 0 {
		return f
	}
	e.millerLoop(f)
	e.finalExp(f)
	return f
}

// Check computes pairing and checks if result is equal to one
func (e *Engine) Check() bool {
	return e.calculate().isOne()
}

// Result computes pairing and returns target group element as result.
func (e *Engine) Result() *E {
	r := e.calculate()
	e.Reset()
	return r
}

// GT returns target group instance.
func (e *Engine) GT() *GT {
	return NewGT()
}
