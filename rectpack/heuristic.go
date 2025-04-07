package rectpack

import (
	"errors"
	"strings"
)

// Heuristic 是一个用于配置矩形打包算法的位字段，包括算法的
// 基本类型、箱体选择方法以及如何分割空白区域的策略。可以通过
// 对特定值进行异或(XOR)运算来实现所需的行为。
//
// 注意并非所有组合都是有效的，此类型的每个常量都会指明其
// 可以与哪些值组合使用。如有疑问，建议直接使用预设值。
//
// 要测试某个值是否有效，请使用 Validate 函数，它会返回一个描述
// 具体问题的错误信息。当使用了无效值时，算法会使用默认值，
// 但不会产生其他错误。
type Heuristic uint16

const (
	/**********************************************************************************************
	* Algorithm types
	**********************************************************************************************/

	// MaxRects selects the MaxRects algorithm for packing. This generally results in the
	// most efficiently packed results when packing to a static size. It can result in a lot of
	// waste if a sensible size and bin heustic is not chosen for the given inputs, but has the
	// most potential for efficiency.
	//
	// Type: Algorithm
	MaxRects Heuristic = 0x0

	// Skyline selects the Skyline algorithm for packing. Skyline provides a good balance between
	// speed and efficiency, and is good for maintaining the least amount of waste at any given
	// time, making it a good choice for dynamic data and simply using whatever the final size may
	// be.
	//
	// Type: Algorithm
	Skyline = 0x1

	// Guillotine selects the Guillotine algorithm for packing. This algorithm is typically
	// faster, but is much more sensitive to choosing the correct packing/splitting methods for
	// specific inputs. This makes it less "general-purpose", but can still be the best choice
	// in certain situations where the input sizes are predictable.
	//
	// Type: Algorithm
	Guillotine = 0x2

	/**********************************************************************************************
	* Bin-Selection
	**********************************************************************************************/

	// BestShortSideFit (BSSF) positions the rectangle against the short side of a free rectangle
	// into which it fits the best.
	//
	//	* Type: Bin-Selection
	//	* Valid With: MaxRects, Guillotine
	BestShortSideFit = 0x00
	// BestLongSideFit (BLSF) positions the rectangle against the long side of a free rectangle
	// into which it fits the best.
	//
	//	* Type: Bin-Selection
	//	* Valid With: MaxRects, Guillotine
	BestLongSideFit = 0x10
	// BestAreaFit (BAF) positions the rectangle into the smallest free rect into which it fits.
	//
	//	* Type: Bin-Selection
	//	* Valid With: MaxRects, Guillotine
	BestAreaFit = 0x20
	// BottomLeft (BL) does the Tetris placement.
	//
	//	* Type: Bin-Selection
	//	* Valid With: MaxRects, Skyline
	BottomLeft = 0x30
	// ContactPoint (CP) choosest the placement where the rectangle touches other rects as much
	// as possible.
	//
	//	* Type: Bin-Selection
	//	* Valid With: MaxRects
	ContactPoint = 0x40
	// WorstAreaFit (WAF) is the opposite of the BestAreaFit (BAF) heuristic. Contrary to its
	// name, this is not always "worse" with speciifc inputs.
	//
	//	* Type: Bin-Selection
	//	* Valid With: Guillotine
	WorstAreaFit = 0x50
	// WorstShortSideFit (WSSF) is the opposite of the BestShortSideFit (BSSF) heuristic. Contrary
	// to its name, this is not always "worse" with speciifc inputs.
	//
	//	* Type: Bin-Selection
	//	* Valid With: Guillotine
	WorstShortSideFit = 0x60
	// WorstLongSideFit (WLSF) is the opposite of the BestLongSideFit (BLSF) heuristic. Contrary
	// to its name, this is not always "worse" with speciifc inputs.
	//
	//	* Type: Bin-Selection
	//	* Valid With: Guillotine
	WorstLongSideFit = 0x70
	// MinWaste (MW) uses a "waste map" to split empty spaces and determine which placement will
	// result in the least amount of wasted space. This is most effective when flip/rotate is
	// enabled by the packer.
	//
	//	* Type: Bin-Selection
	//	* Valid With: Skyline
	MinWaste = 0x80

	/**********************************************************************************************
	* Splitting algorithms (only used with guillotine algorithms)
	**********************************************************************************************/

	// SplitShorterLeftoverAxis (SLAS)
	//
	//	* Type: Split Method
	//	* Valid With: Guillotine
	SplitShorterLeftoverAxis = 0x0000

	// SplitLongerLeftoverAxis (LLAS)
	//
	//	* Type: Split Method
	//	* Valid With: Guillotine
	SplitLongerLeftoverAxis = 0x0100

	// SplitMinimizeArea (MINAS) try to make a single big rectangle at the expense of making the
	// other small.
	//
	//	* Type: Split Method
	//	* Valid With: Guillotine
	SplitMinimizeArea = 0x0200

	// SplitMaximizeArea (MAXAS) try to make both remaining rectangles as even-sized as possible.
	//
	//	*Type: Split Method
	//	* Valid With: Guillotine
	SplitMaximizeArea = 0x0300

	// SplitShorterAxis (SAS)
	//
	//	* Type: Split Method
	//	* Valid With: Guillotine
	SplitShorterAxis = 0x0400

	// SplitLongerAxis (LAS)
	//
	//	* Type: Split Method
	//	* Valid With: Guillotine
	SplitLongerAxis = 0x0500

	/**********************************************************************************************
	* Masks for extracting relevant bits
	**********************************************************************************************/

	typeMask  = 0x000F
	fitMask   = 0x00F0
	splitMask = 0x0F00

	/**********************************************************************************************
	* Present combinations of valid heuristics
	**********************************************************************************************/

	MaxRectsBSSF    = MaxRects | BestShortSideFit
	MaxRectsBL      = MaxRects | BottomLeft
	MaxRectsCP      = MaxRects | ContactPoint
	MaxRectsBLSF    = MaxRects | BestLongSideFit
	MaxRectsBAF     = MaxRects | BestAreaFit
	GuillotineBAF   = Guillotine | BestAreaFit
	GuillotineBSSF  = Guillotine | BestShortSideFit
	GuillotineBLSF  = Guillotine | BestLongSideFit
	GuillotineWAF   = Guillotine | WorstAreaFit
	GuillotineWSSF  = Guillotine | WorstShortSideFit
	GuillotineWLSF  = Guillotine | WorstLongSideFit
	SkylineBLF      = Skyline | BottomLeft
	SkylineMinWaste = Skyline | MinWaste
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

// Validate tests whether the combination of heuristics are in good form. A value of nil is
// returned upon success, otherwise an error with message explaining the error.
//
// Note that invalid heuristics will silently fail and cause the packer to revert to its default
// for that setting.
func (e Heuristic) Validate() error {
	bin := e & fitMask
	split := e & splitMask

	switch e & typeMask {
	case MaxRects:
		if split != 0 {
			return splitErr
		}
		switch bin {
		case BestShortSideFit, BestAreaFit, BottomLeft, ContactPoint, BestLongSideFit:
		default:
			return binErr
		}
	case Skyline:
		if split != 0 {
			return splitErr
		}
		switch bin {
		case BottomLeft, MinWaste:
		default:
			return binErr
		}
	case Guillotine:
		switch split {
		case SplitShorterLeftoverAxis, SplitLongerLeftoverAxis, SplitMinimizeArea, SplitMaximizeArea, SplitShorterAxis, SplitLongerAxis:
		default:
			return splitErr
		}
		switch bin {
		case BestShortSideFit, BottomLeft, ContactPoint, BestLongSideFit, BestAreaFit:
		default:
			return splitErr
		}
	default:
		return algoErr
	}

	return nil
}

// String returns the string representation of the heuristic.
func (e Heuristic) String() string {
	var sb strings.Builder
	var split, bin string

	switch e & typeMask {
	case MaxRects:
		sb.WriteString("MaxRects")
	case Skyline:
		sb.WriteString("Skyline")
	case Guillotine:
		sb.WriteString("Guillotine")
		switch e & splitMask {
		case SplitShorterLeftoverAxis:
			split = "-SLAS"
		case SplitLongerLeftoverAxis:
			split = "-LLAS"
		case SplitMinimizeArea:
			split = "-MINAS"
		case SplitMaximizeArea:
			split = "-MAXAS"
		case SplitShorterAxis:
			split = "-SAS"
		case SplitLongerAxis:
			split = "-LAS"
		}
	}

	switch e & fitMask {
	case BestShortSideFit:
		bin = "BSSF"
	case BestLongSideFit:
		bin = "BLSF"
	case BestAreaFit:
		bin = "BAF"
	case BottomLeft:
		bin = "BL"
	case ContactPoint:
		bin = "CP"
	case WorstAreaFit:
		bin = "WAF"
	case WorstShortSideFit:
		bin = "WSSF"
	case WorstLongSideFit:
		bin = "WLSF"
	case MinWaste:
		bin = "MW"
	}

	if bin != "" {
		if sb.Len() > 0 {
			sb.WriteRune('-')
		}
		sb.WriteString(bin)
	}

	if split != "" {
		sb.WriteRune('-')
		sb.WriteString(split)
	}
	return sb.String()
}
