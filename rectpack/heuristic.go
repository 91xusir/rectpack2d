package rectpack

import (
	"errors"
)

type Heuristic uint16

const (
	MaxRects                 Heuristic = 0x0
	Guillotine                         = 0x2
	BestShortSideFit                   = 0x00
	BestLongSideFit                    = 0x10
	BestAreaFit                        = 0x20
	BottomLeft                         = 0x30
	ContactPoint                       = 0x40
	WorstAreaFit                       = 0x50
	WorstShortSideFit                  = 0x60
	WorstLongSideFit                   = 0x70
	SplitShorterLeftoverAxis           = 0x0000
	SplitLongerLeftoverAxis            = 0x0100
	SplitMinimizeArea                  = 0x0200
	SplitMaximizeArea                  = 0x0300
	SplitShorterAxis                   = 0x0400
	SplitLongerAxis                    = 0x0500

	typeMask  = 0x000F
	fitMask   = 0x00F0
	splitMask = 0x0F00

	/**********************************************************************************************
	* Present combinations of valid heuristics
	**********************************************************************************************/
	MaxRectsBSSF   = MaxRects | BestShortSideFit
	MaxRectsBL     = MaxRects | BottomLeft
	MaxRectsCP     = MaxRects | ContactPoint
	MaxRectsBLSF   = MaxRects | BestLongSideFit
	MaxRectsBAF    = MaxRects | BestAreaFit
	GuillotineBAF  = Guillotine | BestAreaFit
	GuillotineBSSF = Guillotine | BestShortSideFit
	GuillotineBLSF = Guillotine | BestLongSideFit
	GuillotineWAF  = Guillotine | WorstAreaFit
	GuillotineWSSF = Guillotine | WorstShortSideFit
	GuillotineWLSF = Guillotine | WorstLongSideFit
)

// Algorithm returns the algorithm portion of the bitmask.
func (e Heuristic) Algorithm() Heuristic {
	return e & typeMask
}

// Bin returns the bin selection method portion of the bitmask.
func (e Heuristic) Bin() Heuristic {
	return e & fitMask
}

// Split returns the split method portion of the bitmask.
func (e Heuristic) Split() Heuristic {
	return e & splitMask
}

var (
	algoErr  = errors.New("invalid algorithm type specified")
	splitErr = errors.New("split method heuristic is invalid for algorithm type and will be ignored")
	binErr   = errors.New("bin method heuristic is invalid for algorithm type")
)


func ResolveAlgorithm(algo, variant string) Heuristic {
	switch algo {
	case "MaxRects":
		switch variant {
		case "BestShortSideFit":
			return MaxRectsBSSF
		case "BottomLeft":
			return MaxRectsBL
		case "ContactPoint":
			return MaxRectsCP
		case "BestLongSideFit":
			return MaxRectsBLSF
		case "BestAreaFit":
			return MaxRectsBAF
		}
	case "Guillotine":
		switch variant {
		case "BestAreaFit":
			return GuillotineBAF
		case "BestShortSideFit":
			return GuillotineBSSF
		case "BestLongSideFit":
			return GuillotineBLSF
		case "WorstAreaFit":
			return GuillotineWAF
		case "WorstShortSideFit":
			return GuillotineWSSF
		case "WorstLongSideFit":
			return GuillotineWLSF
		}
	}
	return 99
}
