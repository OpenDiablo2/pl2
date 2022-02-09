package pkg

import (
	"fmt"
	"image/color"
	"io"
)

func (pl2 *PL2) Encode(w io.Writer) error {
	if err := pl2.encodeBasePalette(w); err != nil {
		return err
	}

	if err := pl2.encodeTransforms(w); err != nil {
		return err
	}

	if err := pl2.encodeTextColors(w); err != nil {
		return err
	}

	return pl2.encodeTextColorTransforms(w)
}

func (pl2 *PL2) encodeColors(w io.Writer, src color.Palette, colorBytes int) error {
	for idx := range src {
		r, g, b, _ := src[idx].RGBA()

		if colorBytes == 3 {
			if _, err := w.Write([]byte{byte(r), byte(g), byte(b)}); err != nil {
				return fmt.Errorf("could not encode colors, %w", err)
			}
		} else {
			if _, err := w.Write([]byte{byte(r), byte(g), byte(b), 0}); err != nil {
				return fmt.Errorf("could not encode colors, %w", err)
			}
		}
	}

	return nil
}

func (pl2 *PL2) encodeBasePalette(w io.Writer) error {
	pl2.SetMainPalette(pl2.BasePalette) // if nil, generates default

	return pl2.encodeColors(w, pl2.BasePalette, 4)
}

func (pl2 *PL2) encodeTextColors(w io.Writer) error {
	pl2.SetTextPalette(pl2.TextColors) // if nil, generates default

	return pl2.encodeColors(w, pl2.TextColors, 3)
}

func (pl2 *PL2) encodeTransforms(w io.Writer) (err error) {
	if err = pl2.encodeLightingTransforms(w); err != nil {
		return err
	}

	if err = pl2.encodeBlendModeTransforms(w); err != nil {
		return err
	}

	if err = pl2.encodeColorVariationTransforms(w); err != nil {
		return err
	}

	return pl2.encodeOtherTransforms(w)
}

func (pl2 *PL2) encodeLightingTransforms(w io.Writer) (err error) {
	if err = pl2.encodeTransformMulti(w, &pl2.LightLevelVariations); err != nil {
		return err
	}

	if err = pl2.encodeTransformMulti(w, &pl2.InvColorVariations); err != nil {
		return err
	}

	return pl2.encodeTransformSingle(w, &pl2.SelectedUnitShift)
}

func (pl2 *PL2) encodeBlendModeTransforms(w io.Writer) (err error) {
	for blendIdx := range pl2.AlphaBlend {
		if err = pl2.encodeTransformMulti(w, &pl2.AlphaBlend[blendIdx]); err != nil {
			return err
		}
	}

	if err = pl2.encodeTransformMulti(w, &pl2.AdditiveBlend); err != nil {
		return err
	}

	return pl2.encodeTransformMulti(w, &pl2.MultiplicativeBlend)
}

func (pl2 *PL2) encodeColorVariationTransforms(w io.Writer) (err error) {
	if err = pl2.encodeTransformMulti(w, &pl2.HueVariations); err != nil {
		return err
	}

	if err = pl2.encodeTransformSingle(w, &pl2.RedTones); err != nil {
		return err
	}

	if err = pl2.encodeTransformSingle(w, &pl2.GreenTones); err != nil {
		return err
	}

	return pl2.encodeTransformSingle(w, &pl2.BlueTones)
}

func (pl2 *PL2) encodeOtherTransforms(w io.Writer) (err error) {
	if err = pl2.encodeTransformMulti(w, &pl2.UnknownVariations); err != nil {
		return err
	}

	if err = pl2.encodeTransformMulti(w, &pl2.MaxComponentBlend); err != nil {
		return err
	}

	return pl2.encodeTransformSingle(w, &pl2.DarkenedColorShift)
}

func (pl2 *PL2) encodeTransformMulti(w io.Writer, src *[]Transform) error {
	for idx := range *src {
		err := pl2.encodeTransformSingle(w, &((*src)[idx]))

		if err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

func (pl2 *PL2) encodeTransformSingle(w io.Writer, src *Transform) error {
	for idx := range src {
		if _, err := w.Write([]byte{src[idx]}); err != nil {
			return fmt.Errorf("could not encode transform, %w", err)
		}
	}

	return nil
}

func (pl2 *PL2) encodeTextColorTransforms(w io.Writer) (err error) {
	return pl2.encodeTransformMulti(w, &pl2.TextColorShifts)
}
