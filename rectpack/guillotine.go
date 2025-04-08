package rectpack

import (
	"math"
	"slices"
)

type scoreFunc func(width, height int, freeRect *Rect2D) int

type guillotinePack struct {
	algorithmBase
	Merge       bool
	splitMethod Heuristic
	scoreRect   scoreFunc
	freeRects   []Rect2D
}

func newGuillotine(width, height int, heuristic Heuristic) *guillotinePack {
	var packer guillotinePack
	packer.idMaptoRotateCount = make(map[int]int)

	packer.Merge = true
	packer.splitMethod = SplitMinimizeArea
	switch heuristic & fitMask {
	case BestShortSideFit:
		packer.scoreRect = scoreBestShort
	case BestLongSideFit:
		packer.scoreRect = scoreBestLong
	case WorstAreaFit:
		packer.scoreRect = func(w, h int, r *Rect2D) int { return -scoreBestArea(w, h, r) }
	case WorstShortSideFit:
		packer.scoreRect = func(w, h int, r *Rect2D) int { return -scoreBestShort(w, h, r) }
	case WorstLongSideFit:
		packer.scoreRect = func(w, h int, r *Rect2D) int { return -scoreBestLong(w, h, r) }
	default:
		packer.scoreRect = scoreBestArea
	}
	packer.splitMethod = heuristic & splitMask
	packer.Reset(width, height)
	return &packer
}

func (p *guillotinePack) Reset(width, height int) {
	p.algorithmBase.Reset(width, height)
	p.freeRects = p.freeRects[:0]
	p.freeRects = append(p.freeRects, NewRect(0, 0, p.maxWidth, p.maxHeight))
}

func (p *guillotinePack) Insert(padding int, sizes ...Size2D) []Size2D {
	bestFreeRect := 0
	bestRect := 0
	bestFlipped := false
	for len(sizes) > 0 {
		bestScore := math.MaxInt
		for i, freeRect := range p.freeRects {
			for j, size := range sizes {
				padSize(&size, padding)
				if size.Width == freeRect.Width && size.Height == freeRect.Height {
					bestFreeRect = i
					bestRect = j
					bestFlipped = false
					bestScore = math.MinInt
					i = len(p.freeRects)
					break
				} else if p.allowRotate && size.Height == freeRect.Width && size.Width == freeRect.Height {
					bestFreeRect = i
					bestRect = j
					bestFlipped = true
					bestScore = math.MinInt
					i = len(p.freeRects)
					break
				} else if size.Width <= freeRect.Width && size.Height <= freeRect.Height {
					score := p.scoreRect(size.Width, size.Height, &freeRect)
					if score < bestScore {
						bestFreeRect = i
						bestRect = j
						bestFlipped = false
						bestScore = score
					}
				} else if p.allowRotate && size.Height <= freeRect.Width && size.Width <= freeRect.Height {
					score := p.scoreRect(size.Height, size.Width, &freeRect)
					if score < bestScore {
						bestFreeRect = i
						bestRect = j
						bestFlipped = true
						bestScore = score
					}
				}
			}
		}
		if bestScore == math.MaxInt {
			break
		}
		newNode := Rect2D{
			Point2D: p.freeRects[bestFreeRect].Point2D,
			Size2D:  sizes[bestRect],
		}
		if bestFlipped {
			newNode.Width, newNode.Height = newNode.Height, newNode.Width
			if !newNode.IsRotated {
				newNode.RotatedCount++
			}
			newNode.IsRotated = true
		} else {
			if newNode.IsRotated {
				newNode.RotatedCount++
			}
			newNode.IsRotated = false
		}
		p.splitByHeuristic(&p.freeRects[bestFreeRect], &newNode)
		p.freeRects = slices.Delete(p.freeRects, bestFreeRect, bestFreeRect+1)
		sizes = slices.Delete(sizes, bestRect, bestRect+1)
		if p.Merge {
			p.mergeFreeList()
		}
		p.usedArea += newNode.Area()
		unpadRect(&newNode, padding)
		p.idMaptoRotateCount[newNode.ID]+=newNode.RotatedCount
		p.packed = append(p.packed, newNode)
	}
	return sizes
}

func scoreBestArea(width, height int, freeRect *Rect2D) int {
	return freeRect.Width*freeRect.Height - width*height
}

func scoreBestShort(width, height int, freeRect *Rect2D) int {
	leftoverHoriz := abs(freeRect.Width - width)
	leftoverVert := abs(freeRect.Height - height)
	return min(leftoverHoriz, leftoverVert)
}

func scoreBestLong(width, height int, freeRect *Rect2D) int {
	leftoverHoriz := abs(freeRect.Width - width)
	leftoverVert := abs(freeRect.Height - height)
	return max(leftoverHoriz, leftoverVert)
}

func (p *guillotinePack) splitAlongAxis(freeRect, placedRect *Rect2D, splitHorizontal bool) {
	var bottom Rect2D
	bottom.X = freeRect.X
	bottom.Y = freeRect.Y + placedRect.Height
	bottom.Height = freeRect.Height - placedRect.Height
	var right Rect2D
	right.X = freeRect.X + placedRect.Width
	right.Y = freeRect.Y
	right.Width = freeRect.Width - placedRect.Width
	if splitHorizontal {
		bottom.Width = freeRect.Width
		right.Height = placedRect.Height
	} else {
		bottom.Width = placedRect.Width
		right.Height = freeRect.Height
	}
	if bottom.Width > 0 && bottom.Height > 0 {
		p.freeRects = append(p.freeRects, bottom)
	}
	if right.Width > 0 && right.Height > 0 {
		p.freeRects = append(p.freeRects, right)
	}
}

func (p *guillotinePack) findPosition(width, height int, nodeIndex *int) Rect2D {
	var bestNode Rect2D
	bestScore := math.MaxInt
	for i, freeRect := range p.freeRects {
		if width == freeRect.Width && height == freeRect.Height {
			bestNode.X = freeRect.X
			bestNode.Y = freeRect.Y
			bestNode.Width = width
			bestNode.Height = height
			bestScore = math.MinInt
			*nodeIndex = i
			break
		} else if p.allowRotate && height == freeRect.Width && width == freeRect.Height {
			bestNode.X = freeRect.X
			bestNode.Y = freeRect.Y
			bestNode.Width = height
			bestNode.Height = width
			bestScore = math.MinInt
			*nodeIndex = i
			break
		} else if width <= freeRect.Width && height <= freeRect.Height {
			score := p.scoreRect(width, height, &freeRect)
			if score < bestScore {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = width
				bestNode.Height = height
				bestScore = score
				*nodeIndex = i
			}
		} else if p.allowRotate && height <= freeRect.Width && width <= freeRect.Height {
			score := p.scoreRect(height, width, &freeRect)
			if score < bestScore {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = height
				bestNode.Height = width
				bestScore = score
				*nodeIndex = i
			}
		}
	}
	return bestNode
}

func (p *guillotinePack) splitByHeuristic(freeRect, placedRect *Rect2D) {
	w := freeRect.Width - placedRect.Width
	h := freeRect.Height - placedRect.Height
	var splitHorizontal bool
	switch p.splitMethod {
	case SplitShorterLeftoverAxis:
		splitHorizontal = w <= h
	case SplitLongerLeftoverAxis:
		splitHorizontal = w > h
	case SplitMinimizeArea:
		splitHorizontal = placedRect.Width*h > w*placedRect.Height
	case SplitMaximizeArea:
		splitHorizontal = placedRect.Width*h <= w*placedRect.Height
	case SplitShorterAxis:
		splitHorizontal = freeRect.Width <= freeRect.Height
	case SplitLongerAxis:
		splitHorizontal = freeRect.Width > freeRect.Height
	default:
		splitHorizontal = true
	}

	p.splitAlongAxis(freeRect, placedRect, splitHorizontal)
}

func (p *guillotinePack) mergeFreeList() {
	for i := 0; i < len(p.freeRects); i++ {
		for j := i + 1; j < len(p.freeRects); j++ {
			if p.freeRects[i].Width == p.freeRects[i].Width && p.freeRects[i].X == p.freeRects[i].X {
				if p.freeRects[i].Y == p.freeRects[i].Y+p.freeRects[i].Height {
					p.freeRects[i].Y -= p.freeRects[i].Height
					p.freeRects[i].Height += p.freeRects[i].Height
					p.freeRects = slices.Delete(p.freeRects, j, j+1)
					j--
				} else if p.freeRects[i].Y+p.freeRects[i].Height == p.freeRects[i].Y {
					p.freeRects[i].Height += p.freeRects[i].Height
					p.freeRects = slices.Delete(p.freeRects, j, j+1)
					j--
				}
			} else if p.freeRects[i].Height == p.freeRects[i].Height && p.freeRects[i].Y == p.freeRects[i].Y {
				if p.freeRects[i].X == p.freeRects[i].X+p.freeRects[i].Width {
					p.freeRects[i].X -= p.freeRects[i].Width
					p.freeRects[i].Width += p.freeRects[i].Width
					p.freeRects = slices.Delete(p.freeRects, j, j+1)
					j--
				} else if p.freeRects[i].X+p.freeRects[i].Width == p.freeRects[i].X {
					p.freeRects[i].Width += p.freeRects[i].Width
					p.freeRects = slices.Delete(p.freeRects, j, j+1)
					j--
				}
			}
		}
	}
}
