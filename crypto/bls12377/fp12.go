package bls12377

import (
	"errors"
	"math/big"
)

type fp12 struct {
	fp12temp
	fp6 *fp6
}

type fp12temp struct {
	t2  [9]*fe2
	t6  [5]*fe6
	t12 [4]*fe12
}

func newFp12Temp() fp12temp {
	t2 := [9]*fe2{}
	t6 := [5]*fe6{}
	t12 := [4]*fe12{}
	for i := 0; i < len(t2); i++ {
		t2[i] = &fe2{}
	}
	for i := 0; i < len(t6); i++ {
		t6[i] = &fe6{}
	}
	for i := 0; i < len(t12); i++ {
		t12[i] = &fe12{}
	}
	return fp12temp{t2, t6, t12}
}

func newFp12(fp6 *fp6) *fp12 {
	t := newFp12Temp()
	if fp6 == nil {
		return &fp12{t, newFp6(nil)}
	}
	return &fp12{t, fp6}
}

func (e *fp12) fp2() *fp2 {
	return e.fp6.fp2
}

func (e *fp12) fromBytes(in []byte) (*fe12, error) {
	if len(in) != 12*FE_BYTE_SIZE {
		return nil, errors.New("input string length must be equal to 576 bytes")
	}
	fp6 := e.fp6
	c0, err := fp6.fromBytes(in[:6*FE_BYTE_SIZE])
	if err != nil {
		return nil, err
	}
	c1, err := fp6.fromBytes(in[6*FE_BYTE_SIZE:])
	if err != nil {
		return nil, err
	}
	return &fe12{*c0, *c1}, nil
}

func (e *fp12) toBytes(a *fe12) []byte {
	fp6 := e.fp6
	out := make([]byte, 12*FE_BYTE_SIZE)
	copy(out[:6*FE_BYTE_SIZE], fp6.toBytes(&a[0]))
	copy(out[6*FE_BYTE_SIZE:], fp6.toBytes(&a[1]))
	return out
}

func (e *fp12) new() *fe12 {
	return new(fe12)
}

func (e *fp12) zero() *fe12 {
	return new(fe12)
}

func (e *fp12) one() *fe12 {
	return new(fe12).one()
}

func (e *fp12) add(c, a, b *fe12) {
	// c0 = a0 + b0
	// c1 = a1 + b1
	fp6 := e.fp6
	fp6.add(&c[0], &a[0], &b[0])
	fp6.add(&c[1], &a[1], &b[1])
}

func (e *fp12) double(c, a *fe12) {
	// c0 = 2a0
	// c1 = 2a1
	fp6 := e.fp6
	fp6.double(&c[0], &a[0])
	fp6.double(&c[1], &a[1])
}

func (e *fp12) sub(c, a, b *fe12) {
	// c0 = a0 - b0
	// c1 = a1 - b1
	fp6 := e.fp6
	fp6.sub(&c[0], &a[0], &b[0])
	fp6.sub(&c[1], &a[1], &b[1])
}

func (e *fp12) neg(c, a *fe12) {
	// c0 = -a0
	// c1 = -a1
	fp6 := e.fp6
	fp6.neg(&c[0], &a[0])
	fp6.neg(&c[1], &a[1])
}

func (e *fp12) conjugate(c, a *fe12) {
	// c0 = a0
	// c1 = -a1
	fp6 := e.fp6
	c[0].set(&a[0])
	fp6.neg(&c[1], &a[1])
}

func (e *fp12) square(c, a *fe12) {
	// Multiplication and Squaring on Pairing-Friendly Fields
	// Complex squaring algorithm
	// https://eprint.iacr.org/2006/471

	fp6, t := e.fp6, e.t6

	fp6.add(t[0], &a[0], &a[1])      // a0 + a1
	fp6.mul(t[2], &a[0], &a[1])      // v0 = a0a1
	fp6.mulByNonResidue(t[1], &a[1]) // ??a1
	fp6.add(t[1], t[1], &a[0])       // a0 + ??a1
	fp6.double(&c[1], t[2])          // c1 = 2v0
	fp6.mulByNonResidue(t[3], t[2])  // ??a0a1
	fp6.mul(t[0], t[0], t[1])        // (a0 + a1)(a0 + ??a1)
	fp6.sub(t[0], t[0], t[2])        // (a0 + a1)(a0 + ??a1) - v0
	fp6.sub(&c[0], t[0], t[3])       // c0 = (a0 + a1)(a0 + ??a1) - v0
}

func (e *fp12) cyclotomicSquare(c, a *fe12) {
	t, fp2 := e.t2, e.fp2()
	e.fp4Square(t[3], t[4], &a[0][0], &a[1][1])
	fp2.sub(t[2], t[3], &a[0][0])
	fp2.double(t[2], t[2])
	fp2.add(&c[0][0], t[2], t[3])
	fp2.add(t[2], t[4], &a[1][1])
	fp2.double(t[2], t[2])
	fp2.add(&c[1][1], t[2], t[4])
	e.fp4Square(t[3], t[4], &a[1][0], &a[0][2])
	e.fp4Square(t[5], t[6], &a[0][1], &a[1][2])
	fp2.sub(t[2], t[3], &a[0][1])
	fp2.double(t[2], t[2])
	fp2.add(&c[0][1], t[2], t[3])
	fp2.add(t[2], t[4], &a[1][2])
	fp2.double(t[2], t[2])
	fp2.add(&c[1][2], t[2], t[4])
	fp2.mulByNonResidue(t[3], t[6])
	fp2.add(t[2], t[3], &a[1][0])
	fp2.double(t[2], t[2])
	fp2.add(&c[1][0], t[2], t[3])
	fp2.sub(t[2], t[5], &a[0][2])
	fp2.double(t[2], t[2])
	fp2.add(&c[0][2], t[2], t[5])
}

func (e *fp12) mul(c, a, b *fe12) {
	// Guide to Pairing Based Cryptography
	// Algorithm 5.16

	t, fp6 := e.t6, e.fp6

	fp6.mul(t[1], &a[0], &b[0])     // v0 = a0b0
	fp6.mul(t[2], &a[1], &b[1])     // v1 = a1b1
	fp6.add(t[0], &a[0], &a[1])     // a0 + a1
	fp6.add(t[3], &b[0], &b[1])     // b0 + b1
	fp6.mul(t[0], t[0], t[3])       // (a0 + a1)(b0 + b1)
	fp6.sub(t[0], t[0], t[1])       // (a0 + a1)(b0 + b1) - v0
	fp6.sub(&c[1], t[0], t[2])      // c1 = (a0 + a1)(b0 + b1) - v0 - v1
	fp6.mulByNonResidue(t[2], t[2]) // ??v1
	fp6.add(&c[0], t[1], t[2])      // c0 = v0 + 4v1
}

func (fp12 *fp12) mul034(a *fe12, b0, b3, b4 *fe2) {
	// Software Implementation of the Optimal Ate Pairing over Barreto???Naehrig Curves
	// Algorithm 21
	// https://eprint.iacr.org/2010/354.pdf

	fp2, fp6, t := fp12.fp2(), fp12.fp6, fp12.t6

	fp6.mul0(t[0], &a[0], b0)      // t0 = a0b0
	fp6.mul01(t[1], &a[1], b3, b4) // t1 = a1b1
	fp2.add(&t[3][0], b0, b3)      // b0 + b10
	fp6.add(t[2], &a[1], &a[0])    // a0 + a1
	// t2 = (b0 + b10)v + b11v^2 + 0v^3
	fp6.mul01(t[2], t[2], &t[3][0], b4) // c1 = t2(a0 + a1s)
	fp6.sub(t[2], t[2], t[0])           // c1 = v1 - t0
	fp6.sub(&a[1], t[2], t[1])          // c1 = v1 - t0 - t1
	fp6.mulByNonResidue(t[1], t[1])     // ??t1
	fp6.add(&a[0], t[0], t[1])          // c0 = t0 + ??t1
}

func (e *fp12) fp4Square(c0, c1, a0, a1 *fe2) {
	t, fp2 := e.t2, e.fp2()
	fp2.square(t[0], a0)
	fp2.square(t[1], a1)
	fp2.mulByNonResidue(t[2], t[1])
	fp2.add(c0, t[2], t[0])
	fp2.add(t[2], a0, a1)
	fp2.square(t[2], t[2])
	fp2.sub(t[2], t[2], t[0])
	fp2.sub(c1, t[2], t[1])
}

func (e *fp12) inverse(c, a *fe12) {
	// Guide to Pairing Based Cryptography
	// Algorithm 5.16

	fp6, t := e.fp6, e.t6

	fp6.square(t[0], &a[0])         // a0^2
	fp6.square(t[1], &a[1])         // a1^2
	fp6.mulByNonResidue(t[1], t[1]) // ??a1^2
	fp6.sub(t[1], t[0], t[1])       // v = (a0^2 - a1^2)
	fp6.inverse(t[0], t[1])         // v = v^-1
	fp6.mul(&c[0], &a[0], t[0])     // c0 = a0v
	fp6.mul(t[0], t[0], &a[1])      //
	fp6.neg(&c[1], t[0])            // c1 = -a1v
}

func (e *fp12) exp(c, a *fe12, s *big.Int) {
	z := e.one()
	for i := s.BitLen() - 1; i >= 0; i-- {
		e.square(z, z)
		if s.Bit(i) == 1 {
			e.mul(z, z, a)
		}
	}
	c.set(z)
}

func (e *fp12) cyclotomicExp(c, a *fe12, s *big.Int) {
	z := e.one()
	for i := s.BitLen() - 1; i >= 0; i-- {
		e.cyclotomicSquare(z, z)
		if s.Bit(i) == 1 {
			e.mul(z, z, a)
		}
	}
	c.set(z)
}

func (e *fp12) frobeniusMap1(a *fe12) {
	fp6, fp2 := e.fp6, e.fp6.fp2
	fp6.frobeniusMap1(&a[0])
	fp6.frobeniusMap1(&a[1])
	fp2.mul(&a[1][0], &a[1][0], &frobeniusCoeffs12[1])
	fp2.mul(&a[1][1], &a[1][1], &frobeniusCoeffs12[1])
	fp2.mul(&a[1][2], &a[1][2], &frobeniusCoeffs12[1])
}

func (e *fp12) frobeniusMap2(a *fe12) {
	fp6, fp2 := e.fp6, e.fp6.fp2
	fp6.frobeniusMap2(&a[0])
	fp6.frobeniusMap2(&a[1])
	fp2.mul(&a[1][0], &a[1][0], &frobeniusCoeffs12[2])
	fp2.mul(&a[1][1], &a[1][1], &frobeniusCoeffs12[2])
	fp2.mul(&a[1][2], &a[1][2], &frobeniusCoeffs12[2])
}

func (e *fp12) frobeniusMap3(a *fe12) {
	fp6, fp2 := e.fp6, e.fp6.fp2
	fp6.frobeniusMap3(&a[0])
	fp6.frobeniusMap3(&a[1])
	fp2.mul(&a[1][0], &a[1][0], &frobeniusCoeffs12[3])
	fp2.mul(&a[1][1], &a[1][1], &frobeniusCoeffs12[3])
	fp2.mul(&a[1][2], &a[1][2], &frobeniusCoeffs12[3])
}
