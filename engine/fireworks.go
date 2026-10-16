package engine

import "math"

// The fireworks over Shadowdale (O0AC:09F4 and helpers): rockets rise from
// the stands and burst into three rings of forty sparks, drawn pixel by
// pixel with the BIOS (INT 10h 0Ch / 0Dh), each pixel saved and put back.

const (
	rocketStartX, rocketStartY = 65, 65
	rocketSteps                = 60
	rocketStepDelay            = 15 // ms
	fireworkChance             = 10000
	fixedShift                 = 32 // positions are kept in 32nds of a pixel
	skyTopY, skyBottomY        = 8, 65
	sparkRings, sparksPerRing  = 3, 40
	sparkStages                = 5
	sparkFlashRegister         = 9
	sparkFlashColour           = 15
	gravityEvery               = 6  // sparks fall one more 32nd every 6 steps
	sparkStepDelay             = 15 // ms; pacing, see CHECKLIST
	twoPi                      = 6.283179999998538
)

// spark = one 23-byte record of the 2760-byte table (p4AE6).
type spark struct {
	x, y     int16                 // +0 the drawn pixel
	nx, ny   int16                 // +4
	fx, fy   int16                 // +8 position in 32nds
	vx, vy   int16                 // +12
	stage    byte                  // +16 1..5, the colour
	under    byte                  // +17 the pixel it covers
	stageEnd [sparkStages + 1]byte // +17+stage; [stage 5] is the unset +22
}

type fireworksState struct {
	sparks [sparkRings + 1][sparksPerRing + 1]spark
	colour [sparkRings + 1][sparkStages + 1]byte // DS:4AEB
	steps  byte                                  // b4AEA
}

// randomReal = Turbo Pascal Random (real): RandSeed advanced, as a
// fraction of 2^32.
func (e *Engine) randomReal() float64 {
	e.RandSeed = e.RandSeed*0x08088405 + 1
	return float64(e.RandSeed) / (1 << 32)
}

// putPixel / getPixel = O0AC:0000 / 0030 (INT 10h 0Ch, 0Dh).
func (e *Engine) putPixel(x, y int, c byte) { e.S.Set(x, y, c) }
func (e *Engine) getPixel(x, y int) byte    { return e.S.Get(x, y) }

func inSky(y int16) bool { return y > skyTopY && y < skyBottomY }

// fireworkFlash = O0AC:0068: palette register 9 flashes white (not EGA).
func (e *Engine) fireworkFlash() {
	if e.videoMode == 0 {
		return
	}
	e.S.SetPalette(sparkFlashRegister, sparkFlashColour)
	e.Flip()
	e.Delay(1)
	e.S.SetPalette(sparkFlashRegister, sparkFlashRegister)
	e.Flip()
}

// fireworks = O0AC:09F4: until a key is pressed after a burst.
func (e *Engine) fireworks() {
	f := &fireworksState{}
	stop := false
	for !stop {
		if e.Random(fireworkChance) >= 1 {
			continue
		}
		var colours [sparkRings + 1]byte
		for i := 1; i <= sparkRings; i++ {
			colours[i] = 1
		}
		n := e.Random(2) + 1
		for k := 1; k <= n; k++ {
			colours[k] = byte(e.Random(5) + 2)
		}
		x, y := int16(rocketStartX), int16(rocketStartY)
		vx := int16(e.Random(20) + 35)
		vy := -int16(e.Random(5) + 50)
		vx0, vy0 := vx, vy
		e.rocket(&x, &y, &vx, &vy, rocketSteps, false)
		e.sparkSetup(f, x, y, vx, vy, colours)
		vx, vy = vx0, vy0
		x, y = rocketStartX, rocketStartY
		e.rocket(&x, &y, &vx, &vy, rocketSteps, true)
		e.burst(f)
		if e.KeyPressed() {
			e.ReadKeyByte()
			stop = true
		}
	}
}

// rocket = O0AC:085E (x, y, vx, vy, steps, draw): the rocket's flight,
// drawn when draw is set; the first pass only finds where it bursts.
func (e *Engine) rocket(x, y, vx, vy *int16, steps int, draw bool) {
	if draw {
		e.fireworkFlash()
	}
	fx, fy := *x*fixedShift, *y*fixedShift
	var under byte
	if draw && inSky(*y) {
		under = e.getPixel(int(*x), int(*y))
	}
	step := func() (int16, int16) {
		fx += *vx
		fy += *vy + 1
		*vy++
		return fx / fixedShift, fy / fixedShift
	}
	for i := 0; i < steps; i++ {
		nx, ny := step()
		if draw && inSky(*y) {
			e.putPixel(int(*x), int(*y), under)
			under = e.getPixel(int(nx), int(ny))
			e.putPixel(int(nx), int(ny), byte(e.Random(7)+8))
			e.Flip()
		}
		e.Delay(rocketStepDelay)
		*x, *y = nx, ny
	}
	if draw && inSky(*y) {
		e.putPixel(int(*x), int(*y), under)
	}
	*x, *y = step()
}

// sparkSetup = O0AC:00B8 (x, y, cx, cy, colours): three rings of sparks
// leave x, y, each ring heading for its own point around cx, cy.
func (e *Engine) sparkSetup(f *fireworksState, x, y, cx, cy int16, colours [sparkRings + 1]byte) {
	f.steps = 0
	for j := 1; j <= sparkRings; j++ {
		for s := 1; s <= sparkStages; s++ {
			f.colour[j][s] = 1
		}
	}
	for j := 1; j <= sparkRings; j++ {
		c := colours[j]
		col := &f.colour[j]
		if e.videoMode == 0 {
			if c > 1 {
				*col = [sparkStages + 1]byte{0, 2, 3, 3, 3, 2}
			}
		} else {
			if c > 1 {
				col[1] = 15
			}
			col[2] = 15
			if c > 1 {
				col[3] = c + 8
			}
			col[4] = c
			col[5] = 1
		}
		life := byte(e.Random(20) + 25)
		if f.steps < life {
			f.steps = life
		}
		fade := byte(e.Random(5) + 5)
		dim := fade + 15
		a, b := e.randomReal()*twoPi, e.randomReal()*twoPi
		rx := int16(cx) + int16(float64(e.Random(10)+24)*math.Cos(a)*math.Cos(b))
		ry := int16(cy) + int16(float64(e.Random(10)+24)*math.Sin(a)*math.Cos(b))
		for i := 1; i <= sparksPerRing; i++ {
			p := &f.sparks[j][i]
			a, b := e.randomReal()*twoPi, e.randomReal()*twoPi
			p.x, p.y = x, y
			p.fx, p.fy = x*fixedShift, y*fixedShift
			p.vx = rx + int16(math.Cos(a)*16.0*math.Cos(b))
			p.vy = ry + int16(math.Sin(a)*16.0*math.Cos(b))
			p.stage = 1
			p.stageEnd[1] = 1
			p.stageEnd[2] = byte(int(fade) + e.Random(7) - 4)
			p.stageEnd[3] = byte(int(dim) + e.Random(11) - 6)
			p.stageEnd[4] = byte(int(life) - e.Random(7))
			p.under = e.getPixel(int(p.x), int(p.y))
		}
	}
}

// stageLimit is the record byte +17+stage the step is compared with.
func (p *spark) stageLimit() byte { return p.stageEnd[p.stage] }

// burst = O0AC:079B: the sparks fly, cool and go out.
func (e *Engine) burst(f *fireworksState) {
	for step := 1; step <= int(f.steps)+1; step++ {
		e.sparkStep(f, byte(step))
	}
	for i := 1; i <= sparksPerRing; i++ {
		for j := 1; j <= sparkRings; j++ {
			if p := &f.sparks[j][i]; inSky(p.y) {
				e.putPixel(int(p.x), int(p.y), p.under)
			}
		}
	}
	e.Flip()
}

// sparkStep = O0AC:04F7 (step): move every spark, then put back what they
// covered and draw them at their new places.
func (e *Engine) sparkStep(f *fireworksState, step byte) {
	fall := step%gravityEvery == 0
	for j := 1; j <= sparkRings; j++ {
		for i := 1; i <= sparksPerRing; i++ {
			p := &f.sparks[j][i]
			p.fx += p.vx
			p.fy += p.vy
			p.nx, p.ny = p.fx/fixedShift, p.fy/fixedShift
			if fall {
				p.vy++
			}
			if p.vx > 0 {
				p.vx--
			} else if p.vx < 0 {
				p.vx++
			}
			if p.stageLimit() < step && p.stage < sparkStages {
				p.stage++
			}
		}
	}
	for j := 1; j <= sparkRings; j++ {
		for i := 1; i <= sparksPerRing; i++ {
			if p := &f.sparks[j][i]; inSky(p.y) {
				e.putPixel(int(p.x), int(p.y), p.under)
			}
		}
	}
	for j := 1; j <= sparkRings; j++ {
		for i := 1; i <= sparksPerRing; i++ {
			p := &f.sparks[j][i]
			p.x, p.y = p.nx, p.ny
			if inSky(p.y) {
				p.under = e.getPixel(int(p.x), int(p.y))
			}
		}
	}
	for j := 1; j <= sparkRings; j++ {
		for i := 1; i <= sparksPerRing; i++ {
			if p := &f.sparks[j][i]; inSky(p.y) {
				e.putPixel(int(p.x), int(p.y), f.colour[j][p.stage])
			}
		}
	}
	if f.sparks[1][1].stage == 2 {
		e.fireworkFlash()
	}
	e.Flip()
	e.Delay(sparkStepDelay)
}
