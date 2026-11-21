// Package qrcode builds a QR symbol for a short http URL.
// Byte mode, error correction L, versions 2 and 3 only — enough for
// http://255.255.255.255:65535/ and a bit more, and small enough to scan
// from the launcher without a library or a network fetch.
package qrcode

// Modules returns a square of black (true) modules, or nil if text is empty
// or longer than version 3 can hold (53 bytes).
func Modules(text string) [][]bool {
	n := len(text)
	if n == 0 || n > 53 {
		return nil
	}
	ver := ver2
	if n > 32 {
		ver = ver3
	}
	bits := pack(text, ver.data)
	ecc := rsRemainder(bits, rsDivisor(ver.ecc))
	bits = append(bits, ecc...)
	raw := bytesToBits(bits)
	for i := 0; i < 7; i++ { // remainder bits, both versions
		raw = append(raw, false)
	}
	base := pattern(ver)
	best, bestP := (*grid)(nil), int(^uint(0)>>1)
	for mask := 0; mask < 8; mask++ {
		g := base.clone()
		place(g, raw, mask)
		drawFormat(g, mask)
		if p := penalty(g); p < bestP {
			best, bestP = g, p
		}
	}
	return best.dark
}

type version struct{ size, data, ecc, align int }

var ver2 = version{25, 34, 10, 18}
var ver3 = version{29, 55, 15, 22}

type grid struct {
	n    int
	dark [][]bool
	fun  [][]bool
}

func newGrid(n int) *grid {
	g := &grid{n: n, dark: make([][]bool, n), fun: make([][]bool, n)}
	for i := range g.dark {
		g.dark[i] = make([]bool, n)
		g.fun[i] = make([]bool, n)
	}
	return g
}

func (g *grid) clone() *grid {
	c := newGrid(g.n)
	for y := 0; y < g.n; y++ {
		copy(c.dark[y], g.dark[y])
		copy(c.fun[y], g.fun[y])
	}
	return c
}

func (g *grid) force(x, y int, black bool) {
	if x < 0 || y < 0 || x >= g.n || y >= g.n {
		return
	}
	g.dark[y][x] = black
	g.fun[y][x] = true
}

func pattern(v version) *grid {
	g := newGrid(v.size)
	finder := func(x0, y0 int) {
		for dy := -1; dy <= 7; dy++ {
			for dx := -1; dx <= 7; dx++ {
				x, y := x0+dx, y0+dy
				edge := dx == 0 || dy == 0 || dx == 6 || dy == 6
				core := dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4
				g.force(x, y, dx >= 0 && dy >= 0 && dx <= 6 && dy <= 6 && (edge || core))
			}
		}
	}
	finder(0, 0)
	finder(v.size-7, 0)
	finder(0, v.size-7)
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			g.force(v.align+dx, v.align+dy, dx == 0 && dy == 0 || abs(dx) == 2 || abs(dy) == 2)
		}
	}
	for i := 0; i < v.size; i++ {
		if !g.fun[6][i] {
			g.force(i, 6, i%2 == 0)
		}
		if !g.fun[i][6] {
			g.force(6, i, i%2 == 0)
		}
	}
	for i := 0; i <= 8; i++ {
		if i != 6 {
			g.fun[i][8] = true
			g.fun[8][i] = true
		}
	}
	for i := 0; i < 8; i++ {
		g.fun[8][v.size-1-i] = true
	}
	for i := 8; i < 15; i++ {
		g.fun[v.size-15+i][8] = true
	}
	g.fun[v.size-8][8] = true
	return g
}

func pack(text string, dataLen int) []byte {
	var buf []byte
	nbits := 0
	add := func(val, n int) {
		for i := n - 1; i >= 0; i-- {
			if nbits%8 == 0 {
				buf = append(buf, 0)
			}
			if val&(1<<i) != 0 {
				buf[len(buf)-1] |= 1 << (7 - nbits%8)
			}
			nbits++
		}
	}
	add(0x4, 4)
	add(len(text), 8)
	for _, c := range []byte(text) {
		add(int(c), 8)
	}
	term := dataLen*8 - nbits
	if term > 4 {
		term = 4
	}
	if term > 0 {
		add(0, term)
	}
	for nbits%8 != 0 {
		add(0, 1)
	}
	pad := 0xEC
	for len(buf) < dataLen {
		add(pad, 8)
		pad ^= 0xEC ^ 0x11
	}
	return buf
}

func bytesToBits(b []byte) []bool {
	out := make([]bool, 0, len(b)*8)
	for _, v := range b {
		for i := 7; i >= 0; i-- {
			out = append(out, v&(1<<uint(i)) != 0)
		}
	}
	return out
}

func place(g *grid, bits []bool, mask int) {
	i := 0
	for right := g.n - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < g.n; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				y := vert
				if (right+1)&2 == 0 {
					y = g.n - 1 - vert
				}
				if g.fun[y][x] {
					continue
				}
				bit := false
				if i < len(bits) {
					bit = bits[i]
					i++
				}
				if masked(mask, x, y) {
					bit = !bit
				}
				g.dark[y][x] = bit
			}
		}
	}
}

func masked(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return x*y%2+x*y%3 == 0
	case 6:
		return (x*y%2+x*y%3)%2 == 0
	default:
		return ((x+y)%2+x*y%3)%2 == 0
	}
}

func drawFormat(g *grid, mask int) {
	bits := formatBits(mask)
	bit := func(i int) bool { return bits>>uint(i)&1 == 1 }
	for i := 0; i <= 5; i++ {
		g.force(8, i, bit(i))
	}
	g.force(8, 7, bit(6))
	g.force(8, 8, bit(7))
	g.force(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		g.force(14-i, 8, bit(i))
	}
	for i := 0; i < 8; i++ {
		g.force(g.n-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		g.force(8, g.n-15+i, bit(i))
	}
	g.force(8, g.n-8, true)
}

func formatBits(mask int) int {
	data := (1 << 3) | mask // ECC level L
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	return ((data << 10) | rem) ^ 0x5412
}

func penalty(g *grid) int {
	p, n := 0, g.n
	run := func(at func(i int) bool) {
		count, prev := 1, at(0)
		for i := 1; i < n; i++ {
			if at(i) == prev {
				count++
				continue
			}
			if count >= 5 {
				p += 3 + count - 5
			}
			count, prev = 1, at(i)
		}
		if count >= 5 {
			p += 3 + count - 5
		}
	}
	for y := 0; y < n; y++ {
		yy := y
		run(func(x int) bool { return g.dark[yy][x] })
	}
	for x := 0; x < n; x++ {
		xx := x
		run(func(y int) bool { return g.dark[y][xx] })
	}
	for y := 0; y < n-1; y++ {
		for x := 0; x < n-1; x++ {
			c := g.dark[y][x]
			if c == g.dark[y][x+1] && c == g.dark[y+1][x] && c == g.dark[y+1][x+1] {
				p += 3
			}
		}
	}
	dark := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if g.dark[y][x] {
				dark++
			}
		}
	}
	k := dark*100/(n*n) - 50
	if k < 0 {
		k = -k
	}
	return p + (k/5)*10
}

var exp [512]byte
var lg [256]int

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		exp[i] = byte(x)
		lg[x] = i
		x <<= 1
		if x >= 0x100 {
			x ^= 0x11d
		}
	}
	for i := 255; i < 512; i++ {
		exp[i] = exp[i-255]
	}
}

func mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return exp[int(lg[a])+int(lg[b])]
}

func rsDivisor(nsym int) []byte {
	gen := []byte{1}
	for i := 0; i < nsym; i++ {
		next := make([]byte, len(gen)+1)
		for j, c := range gen {
			next[j] ^= c
			next[j+1] ^= mul(c, exp[i])
		}
		gen = next
	}
	return gen
}

func rsRemainder(data, divisor []byte) []byte {
	res := make([]byte, len(divisor)-1)
	for _, b := range data {
		factor := b ^ res[0]
		copy(res, res[1:])
		res[len(res)-1] = 0
		for i, coef := range divisor[1:] {
			res[i] ^= mul(coef, factor)
		}
	}
	return res
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
