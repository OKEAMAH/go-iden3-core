package babyjub

import (
	"fmt"
	"math/big"
)

var Q *big.Int
var A *big.Int
var D *big.Int

var Zero *big.Int
var One *big.Int
var MinusOne *big.Int

var Order *big.Int
var SubOrder *big.Int

var B8 Point

func NewIntFromString(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(fmt.Sprintf("Bad base 10 string %s", s))
	}
	return v
}

func init() {
	Zero = big.NewInt(0)
	One = big.NewInt(1)
	MinusOne = big.NewInt(-1)
	Q = NewIntFromString(
		"21888242871839275222246405745257275088548364400416034343698204186575808495617")
	A = NewIntFromString("168700")
	D = NewIntFromString("168696")

	Order = NewIntFromString(
		"21888242871839275222246405745257275088614511777268538073601725287587578984328")
	SubOrder = new(big.Int).Rsh(Order, 3)

	B8.X = NewIntFromString(
		"17777552123799933955779906779655732241715742912184938656739573121738514868268")
	B8.Y = NewIntFromString(
		"2626589144620713026669568689430873010625803728049924121243784502389097019475")
}

type Point struct {
	X *big.Int
	Y *big.Int
}

func NewPoint() *Point {
	return &Point{X: new(big.Int), Y: new(big.Int)}
}

func (p *Point) Set(c *Point) *Point {
	p.X.Set(c.X)
	p.Y.Set(c.Y)
	return p
}

func (res *Point) Add(a *Point, b *Point) *Point {
	// x = (a.x * b.y + b.x * a.y) * (1 + D * a.x * b.x * a.y * b.y)^-1 mod q
	x1a := new(big.Int).Mul(a.X, b.Y)
	x1b := new(big.Int).Mul(b.X, a.Y)
	x1a.Add(x1a, x1b) // x1a = a.x * b.y + b.x * a.y

	x2 := new(big.Int).Set(D)
	x2.Mul(x2, a.X)
	x2.Mul(x2, b.X)
	x2.Mul(x2, a.Y)
	x2.Mul(x2, b.Y)
	x2.Add(One, x2)
	x2.Mod(x2, Q)
	x2.ModInverse(x2, Q) // x2 = (1 + D * a.x * b.x * a.y * b.y)^-1

	// y = (a.y * b.y + A * a.x * a.x) * (1 - D * a.x * b.x * a.y * b.y)^-1 mod q
	y1a := new(big.Int).Mul(a.Y, b.Y)
	y1b := new(big.Int).Set(A)
	y1b.Mul(y1b, a.X)
	y1b.Mul(y1b, b.X)

	y1a.Sub(y1a, y1b) // y1a = a.y * b.y - A * a.x * b.x

	y2 := new(big.Int).Set(D)
	y2.Mul(y2, a.X)
	y2.Mul(y2, b.X)
	y2.Mul(y2, a.Y)
	y2.Mul(y2, b.Y)
	y2.Sub(One, y2)
	y2.Mod(y2, Q)
	y2.ModInverse(y2, Q) // y2 = (1 - D * a.x * b.x * a.y * b.y)^-1

	res.X = x1a.Mul(x1a, x2)
	res.X = res.X.Mod(res.X, Q)

	res.Y = y1a.Mul(y1a, y2)
	res.Y = res.Y.Mod(res.Y, Q)

	return res
}

func (res *Point) Mul(s *big.Int, p *Point) *Point {
	res.X = big.NewInt(0)
	res.Y = big.NewInt(1)
	exp := NewPoint().Set(p)

	for i := 0; i < s.BitLen(); i++ {
		if s.Bit(i) == 1 {
			res.Add(res, exp)
		}
		exp.Add(exp, exp)
	}

	return res
}

func (p *Point) InCurve() bool {
	x2 := new(big.Int).Set(p.X)
	x2.Mul(x2, x2)
	x2.Mod(x2, Q)

	y2 := new(big.Int).Set(p.Y)
	y2.Mul(y2, y2)
	y2.Mod(y2, Q)

	a := new(big.Int).Mul(A, x2)
	a.Add(a, y2)
	a.Mod(a, Q)

	b := new(big.Int).Set(D)
	b.Mul(b, x2)
	b.Mul(b, y2)
	b.Add(One, b)
	b.Mod(b, Q)

	return a.Cmp(b) == 0
}

func (p *Point) InSubGroup() bool {
	if !p.InCurve() {
		return false
	}
	res := NewPoint().Mul(SubOrder, p)
	return (res.X.Cmp(Zero) == 0) && (res.Y.Cmp(One) == 0)
}

func (p *Point) Compress() [32]byte {
	leBuf := [32]byte{}
	beBuf := p.Y.Bytes()
	for i, b := range beBuf {
		leBuf[len(beBuf)-1-i] = b
	}
	if p.X.Cmp(new(big.Int).Rsh(Q, 1)) == 1 {
		leBuf[31] = leBuf[31] | 0x80
	}
	return leBuf
}

func (p *Point) Decompress(leBuf [32]byte) (*Point, error) {
	sign := One
	if (leBuf[31] & 0x80) != 0x00 {
		sign = MinusOne
		leBuf[31] = leBuf[31] & 0x7F
	}
	beBuf := [32]byte{}
	for i, b := range leBuf {
		beBuf[len(leBuf)-1-i] = b
	}
	p.Y.SetBytes(beBuf[:])
	if p.Y.Cmp(Q) >= 0 {
		return nil, fmt.Errorf("p.y >= Q")
	}

	y2 := new(big.Int).Mul(p.Y, p.Y)
	y2.Mod(y2, Q)
	xa := big.NewInt(1)
	xa.Sub(xa, y2) // xa == 1 - y^2

	xb := new(big.Int).Mul(D, y2)
	xb.Mod(xb, Q)
	xb.Sub(A, xb) // xb = A - d * y^2

	if xb.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("division by 0")
	}
	xb.ModInverse(xb, Q)
	p.X.Mul(xa, xb) // xa / xb
	p.X.Mod(p.X, Q)
	p.X.ModSqrt(p.X, Q)
	p.X.Mul(p.X, sign)
	p.X.Mod(p.X, Q)

	return p, nil
}
