package pkg

import (
	"image/color"
	"math"

	color2 "github.com/lucasb-eyer/go-colorful"
)

func (pl2 *PL2) regenerate() {
	pl2.SetMainPalette(pl2.BasePalette)
	pl2.SetTextPalette(pl2.TextColors)
	pl2.generateTransforms()
	pl2.generateTextColorTransforms()
}

func (pl2 *PL2) generateTransforms() {
	pl2.generateLightingTransforms()
	pl2.generateBlendModeTransforms()
	pl2.generateColorVariationTransforms()
	pl2.generateOtherTransforms()
}

func (pl2 *PL2) generateLightingTransforms() {
	pl2.generateLightLevelVariations()
	pl2.generateInvColorVariations()
	pl2.generateSelectedUnitTransforms()
}

func (pl2 *PL2) generateLightLevelVariations() {
	fnTransform := func(idx int, n uint8) uint8 {
		const divisor = 5
		return uint8((uint32(idx) + 1) * uint32(n) >> divisor)
	}

	pl2.LightLevelVariations = pl2.applyVariations(lightLevelVariations, fnTransform)
}

func (pl2 *PL2) generateInvColorVariations() {
	fnTransform := func(idx int, n uint8) uint8 {
		const reduce = 4

		return uint8((uint32(idx+1)*uint32(math.MaxUint8-n))>>reduce + uint32(n))
	}

	pl2.InvColorVariations = pl2.applyVariations(invColorVariations, fnTransform)
}

func rgba2hsl(c color.Color) color2.Color {
	c2, _ := color2.MakeColor(c)

	return c2
}

func (pl2 *PL2) getHSLColors() []color2.Color {
	if pl2.hslColorsBuffer != nil {
		return pl2.hslColorsBuffer
	}

	hslColors := make([]color2.Color, numPaletteColors)

	for idx := range hslColors {
		hslColors[idx] = rgba2hsl(pl2.BasePalette[idx])
	}

	pl2.hslColorsBuffer = hslColors

	return hslColors
}

func (pl2 *PL2) generateSelectedUnitTransforms() {
	hslColors := pl2.getHSLColors()

	// for each, we increase luminosity by 20%
	const luminosity20pct = 0.2

	for idx := range hslColors {
		h, s, l := hslColors[idx].Hsl()

		if l != 0 {
			l = math.Min(1.0, l+luminosity20pct)
		}

		c := color2.Hsl(h, s, l)

		pl2.SelectedUnitShift[idx] = uint8(pl2.BasePalette.Index(c))
	}
}

func getBlendRatio(blendLevel int) float64 {
	const blendStep25pct float64 = 0.25 // makes the blend step increment 25% per level

	if blendLevel > 3 || blendLevel < 0 {
		blendLevel = 0
	}

	return blendStep25pct * float64(blendLevel+1)
}

func (pl2 *PL2) generateBlendModeTransforms() {
	pl2.generateAlphaTransforms()
	pl2.generateAdditiveTransforms()
	pl2.generateMultiplicativeTransforms()
}

func (pl2 *PL2) generateAlphaTransforms() {
	pl2.AlphaBlend = make([][]Transform, alphaBlendCoarse)

	for blendIdx := range pl2.AlphaBlend {
		pl2.AlphaBlend[blendIdx] = make([]Transform, alphaBlendFine)

		blend := getBlendRatio(blendIdx)
		inverted := 1 - blend

		fn := func(src, dst uint8) uint8 {
			componentA := uint8(inverted * float64(dst))
			componentB := uint8(blend * float64(src))

			return componentA + componentB
		}

		for src := range pl2.BasePalette {
			for dst := range pl2.AlphaBlend[blendIdx] {
				pl2.AlphaBlend[blendIdx][src][dst] = pl2.getClosestBlendIndex(src, dst, fn)
			}
		}
	}
}

func (pl2 *PL2) generateAdditiveTransforms() {
	pl2.AdditiveBlend = make([]Transform, additiveBlends)

	fn := func(src, dst uint8) uint8 {
		sum := int(src) + int(dst)

		if sum > math.MaxUint8 {
			sum = math.MaxUint8
		}

		return uint8(sum)
	}

	for _ = range pl2.AdditiveBlend {
		for dstIndex := range pl2.BasePalette {
			for srcIndex := range pl2.BasePalette {
				pl2.AdditiveBlend[srcIndex][dstIndex] = pl2.getClosestBlendIndex(srcIndex, dstIndex, fn)
			}
		}
	}
}

func (pl2 *PL2) generateMultiplicativeTransforms() {
	pl2.MultiplicativeBlend = make([]Transform, multiplyBlends)

	fn := func(src, dst uint8) uint8 {
		return uint8((float64(src) * float64(dst)) / math.MaxUint8)
	}

	for _ = range pl2.MultiplicativeBlend {
		for dstIndex := range pl2.BasePalette {
			for srcIndex := range pl2.BasePalette {
				pl2.MultiplicativeBlend[dstIndex][srcIndex] = pl2.getClosestBlendIndex(srcIndex, dstIndex, fn)
			}
		}
	}
}

func (pl2 *PL2) generateColorVariationTransforms() {
	pl2.generateHueTransforms()
	pl2.generateRGBTransforms()
}

const (
	hueSteps           int     = 24
	maxDegrees         float64 = 360
	hueRotationPerStep float64 = 15 // degrees, normalized
)

// we're gonna be using normalized values for HSL shit because the library we are using
// implemented hsl with normalized values between 0 and 1.
func (pl2 *PL2) generateHueTransforms() {
	pl2.HueVariations = make([]Transform, hueVariations)

	trsIdx := 0

	for shiftIdx := 0; shiftIdx < hueSteps; shiftIdx++ {
		for palIdx := 0; palIdx < 256; palIdx++ {
			hslColors := pl2.getHSLColors()
			h, s, l := hslColors[palIdx].Hsl()

			h += float64(shiftIdx) * hueRotationPerStep

			if h > maxDegrees {
				h -= maxDegrees
			}

			pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(color2.Hsl(h, s, l)))
		}

		trsIdx++
	}

	for shiftIdx := 0; shiftIdx < hueSteps; shiftIdx++ {
		for palIdx := 0; palIdx < 256; palIdx++ {
			hslColors := pl2.getHSLColors()

			h, s, l := hslColors[palIdx].Hsl()

			h += float64(shiftIdx) * hueRotationPerStep
			if h > maxDegrees {
				h -= maxDegrees
			}

			s = 0.5

			l -= 0.1

			if l < 0 {
				l = 0
			}

			pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(color2.Hsl(h, s, l)))
		}

		trsIdx++
	}

	for shiftIdx := 0; shiftIdx < hueSteps; shiftIdx++ {
		for palIdx := 0; palIdx < 256; palIdx++ {
			hslColors := pl2.getHSLColors()
			h, s, l := hslColors[palIdx].Hsl()

			h += float64(shiftIdx) * hueRotationPerStep
			for h > maxDegrees {
				h -= maxDegrees
			}

			s = 0.5

			l += 0.2

			if l > 1 {
				l = 1
			}

			pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(color2.Hsl(h, s, l)))
		}

		trsIdx++
	}

	hslColors := pl2.getHSLColors()

	for palIdx := 0; palIdx < 256; palIdx++ { // greyscale
		H, S, L := hslColors[palIdx].Hsl()

		S = 0
		L /= 2

		c := color2.Hsl(H, S, L)

		pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(c))
	}

	trsIdx++

	for palIdx := 0; palIdx < 256; palIdx++ {
		h, s, l := hslColors[palIdx].Hsl()

		s = 0
		l += 0.2
		l /= 1.2

		pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(color2.Hsl(h, s, l)))
	}

	trsIdx++

	for shiftIdx := 0; shiftIdx < hueSteps; shiftIdx++ {
		for palIdx := 1; palIdx < 256; palIdx++ {
			hslColors := pl2.getHSLColors()
			h, s, l := hslColors[palIdx].Hsl()

			pl2.HueVariations[trsIdx][palIdx] = pl2.HueVariations[trsIdx-1][palIdx]

			const tolerance = 3 * hueRotationPerStep

			if h > tolerance && h < maxDegrees - tolerance {
				h += float64(shiftIdx) * hueRotationPerStep

				for h > maxDegrees {
					h -= maxDegrees
				}

				pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(color2.Hsl(h, s, l)))
			}
		}

		trsIdx++
	}

	trsIdx++

	// the full saturation variants
	for shiftIdx := 0; shiftIdx < hueSteps/2; shiftIdx++ {
		hslColors = pl2.getHSLColors()
		for palIdx := 0; palIdx < 256; palIdx++ {
			h, s, l := hslColors[palIdx].Hsl()

			h = float64(shiftIdx) * (hueRotationPerStep * 2)

			for h > maxDegrees {
				h -= maxDegrees
			}

			s = 1.0

			pl2.HueVariations[trsIdx][palIdx] = uint8(pl2.BasePalette.Index(color2.Hsl(h, s, l)))
		}

		trsIdx++
	}
}

func (pl2 *PL2) generateRGBTransforms() {
	fn := func(r, g, b float64) color.Color {
		return color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: math.MaxUint8,
		}
	}

	// get magnitude for a color, use for Red, Green, and Blue versions
	for palIdx := 1; palIdx < 256; palIdx++ {
		base := pl2.BasePalette[palIdx]

		r, g, b, _ := base.RGBA()

		rr := float64(r) * float64(r)
		gg := float64(g) * float64(g)
		bb := float64(b) * float64(b)

		m := math.Sqrt(rr + gg + bb) / math.MaxUint8

		pl2.RedTones[palIdx] = uint8(pl2.BasePalette.Index(fn(m, 0, 0)))
		pl2.GreenTones[palIdx] = uint8(pl2.BasePalette.Index(fn(0, m, 0)))
		pl2.BlueTones[palIdx] = uint8(pl2.BasePalette.Index(fn(0, 0, m)))
	}
}

func (pl2 *PL2) generateOtherTransforms() {
	pl2.UnknownVariations = make([]Transform, unknownVariations)

	pl2.generateMaxComponentTransform()
	pl2.generateDarkenedUnitTransform()
}

func (pl2 *PL2) generateMaxComponentTransform() {
	pl2.MaxComponentBlend = make([]Transform, maxComponentBlends)

	fnMax := func(r, g, b uint32) uint32 {
		max := r

		if g > max {
			max = g
		}

		if b > max {
			max = b
		}

		return max
	}

	fnApplyMax := func(src, dst, max uint32) uint8 {
		m := float64(uint8(max)) / math.MaxUint8
		inv := 1 - m
		s := uint8(float64(src) * inv / math.MaxUint8)
		d := uint8(float64(dst) * m / math.MaxUint8)

		return s + d
	}

	for dstIdx := 1; dstIdx < numPaletteColors; dstIdx++ {
		for srcIdx := 1; srcIdx < numPaletteColors; srcIdx++ {
			src := pl2.BasePalette[srcIdx]
			dst := pl2.BasePalette[dstIdx]

			sr, sg, sb, _ := src.RGBA()
			dr, dg, db, _ := dst.RGBA()

			max := fnMax(dr, dg, db)

			blended := color.RGBA{
				R: fnApplyMax(sr, dr, max),
				G: fnApplyMax(sg, dg, max),
				B: fnApplyMax(sb, db, max),
				A: math.MaxUint8,
			}

			pl2.MaxComponentBlend[srcIdx][dstIdx] = uint8(pl2.BasePalette.Index(blended))
		}
	}
}

func (pl2 *PL2) generateDarkenedUnitTransform() {
	fn := func(n uint32) uint8 {
		const third = 3
		return uint8(n / third)
	}

	for colorIndex := range pl2.DarkenedColorShift {
		cidx := uint32(colorIndex)

		r, g, b, _ := pl2.BasePalette[cidx].RGBA()

		// the transform function is applied to each RGB component, per palette entry
		newColor := color.RGBA{
			R: fn(r),
			G: fn(g),
			B: fn(b),
		}

		pl2.DarkenedColorShift[colorIndex] = uint8(pl2.BasePalette.Index(newColor))
	}
}

func defaultTextColors() color.Palette {
	p := make(color.Palette, numTextColors)

	//nolint:gomnd // arcane bullshit from blizzard
	defaults := []color.RGBA{
		{R: 0xFF, G: 0xFF, B: 0xFF, A: math.MaxUint8},
		{R: 0xFF, G: 0x4D, B: 0x4D, A: math.MaxUint8},
		{R: 0x00, G: 0xFF, B: 0x00, A: math.MaxUint8},
		{R: 0x69, G: 0x69, B: 0xFF, A: math.MaxUint8},
		{R: 0xC7, G: 0xB3, B: 0x77, A: math.MaxUint8},
		{R: 0x69, G: 0x69, B: 0x69, A: math.MaxUint8},
		{R: 0x00, G: 0x00, B: 0x00, A: math.MaxUint8},
		{R: 0xD0, G: 0xC2, B: 0x7D, A: math.MaxUint8},
		{R: 0xFF, G: 0xA8, B: 0x00, A: math.MaxUint8},
		{R: 0xFF, G: 0xFF, B: 0x64, A: math.MaxUint8},
		{R: 0x00, G: 0x80, B: 0x00, A: math.MaxUint8},
		{R: 0xAE, G: 0x00, B: 0xFF, A: math.MaxUint8},
		{R: 0x00, G: 0xC8, B: 0x00, A: math.MaxUint8},
	}
	for idx := range defaults {
		p[idx] = defaults[idx]
	}

	return p
}

func (pl2 *PL2) generateTextColorTransforms() {
	pl2.TextColorShifts = make([]Transform, textShifts)

	fn := func(a, b color.Color) color.Color {
		ar, ag, ab, _ := a.RGBA()
		br, _, _, _ := b.RGBA()

		intensity := int(br / math.MaxUint8)

		return color.RGBA{
			R: uint8((int(ar / math.MaxUint8) * intensity) / math.MaxUint8),
			G: uint8((int(ag / math.MaxUint8) * intensity) / math.MaxUint8),
			B: uint8((int(ab / math.MaxUint8) * intensity) / math.MaxUint8),
			A: uint8(math.MaxUint8),
		}
	}

	for textColorIdx := 1; textColorIdx < len(pl2.TextColorShifts); textColorIdx++ {
		textColor := pl2.TextColors[textColorIdx]

		for colorIdx := 1; colorIdx < numPaletteColors; colorIdx++ {
			baseColor := pl2.BasePalette[colorIdx]
			dstColor := fn(textColor, baseColor)

			pl2.TextColorShifts[textColorIdx][colorIdx] = uint8(pl2.BasePalette.Index(dstColor))
		}
	}
}

type simpleTransform = func(idx int, component uint8) uint8

func (pl2 *PL2) applyVariations(numTransforms int, fn simpleTransform) []Transform {
	trs := make([]Transform, numTransforms)

	for variationIndex := range trs {
		quickLookup := make([]*uint8, numPaletteColors)

		for colorIndex := range trs[variationIndex] {
			vidx := variationIndex
			cidx := uint32(colorIndex)

			if quickLookup[cidx] != nil {
				trs[variationIndex][colorIndex] = *quickLookup[cidx]
				continue
			}

			r, g, b, _ := pl2.BasePalette[cidx].RGBA()
			r8 := uint8(r)
			g8 := uint8(g)
			b8 := uint8(b)

			// the transform function is applied to each RGB component, per palette entry
			newColor := color.RGBA{
				R: fn(vidx, r8),
				G: fn(vidx, g8),
				B: fn(vidx, b8),
			}

			transformIdx := uint8(pl2.BasePalette.Index(newColor))
			quickLookup[cidx] = &transformIdx

			trs[variationIndex][colorIndex] = transformIdx
		}
	}

	return trs
}

type blendFn func(componentA, componentB uint8) uint8

func (pl2 *PL2) getClosestBlendIndex(src, dst int, fn blendFn) uint8 {
	sr, sg, sb, _ := pl2.BasePalette[src].RGBA()
	dr, dg, db, _ := pl2.BasePalette[dst].RGBA()

	blended := color.RGBA{
		R: fn(uint8(sr), uint8(dr)),
		G: fn(uint8(sg), uint8(dg)),
		B: fn(uint8(sb), uint8(db)),
		A: math.MaxUint8,
	}

	return uint8(pl2.BasePalette.Index(blended))
}
