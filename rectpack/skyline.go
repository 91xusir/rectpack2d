package rectpack

import (
	"math"
	"slices"
)

type skylineNode struct {
	X, Y, Width int
}

type skylinePack struct {
	algorithmBase
	levelSelect Heuristic
	skyline     []skylineNode
	wasteMap    *guillotinePack
}

func newSkyline(width, height int, heuristic Heuristic) *skylinePack {
	var packer skylinePack
	packer.idMaptoRotateCount = make(map[int]int)

	switch heuristic & fitMask {
	case MinWaste:
		packer.levelSelect = MinWaste
		packer.wasteMap = newGuillotine(width, height, BestAreaFit)
	default: // BottomLeft
		packer.levelSelect = BottomLeft
	}

	packer.Reset(width, height)
	return &packer
}

func (p *skylinePack) Reset(width, height int) {
	p.algorithmBase.Reset(width, height)
	p.skyline = p.skyline[:0]
	p.skyline = append(p.skyline, skylineNode{X: 0, Y: 0, Width: p.maxWidth})

	if p.wasteMap != nil {
		p.wasteMap.Reset(width, height)
	}
}

func (p *skylinePack) Insert(padding int, sizes ...Size2D) []Size2D {
	for len(sizes) > 0 {
		var bestNode Rect2D
		bestScore1 := math.MaxInt
		bestScore2 := math.MaxInt
		bestBinIndex := -1
		bestSizeIndex := -1
		for i, size := range sizes {
			var score1, score2, index int
			var newNode Rect2D
			padSize(&size, padding)
			switch p.levelSelect {
			case MinWaste:
				newNode = p.findMinWaste(size.Width, size.Height, &score2, &score1, &index)
			default: // LevelBottomLeft or invalid
				newNode = p.findBottomLeft(size.Width, size.Height, &score1, &score2, &index)
			}
			if newNode.Height != 0 {
				if score1 < bestScore1 || (score1 == bestScore1 && score2 < bestScore2) {
					bestNode = newNode
					bestScore1 = score1
					bestScore2 = score2
					bestBinIndex = index
					bestSizeIndex = i
				}
			}
		}
		if bestSizeIndex == -1 {
			break
		}
		// Perform the actual packing.
		p.addLevel(bestBinIndex, &bestNode)
		p.usedArea += bestNode.Area()
		unpadRect(&bestNode, padding)
		bestNode.ID = sizes[bestSizeIndex].ID
		p.packed = append(p.packed, bestNode)
		p.idMaptoRotateCount[bestNode.ID] += bestNode.RotatedCount
		sizes = slices.Delete(sizes, bestSizeIndex, bestSizeIndex+1)
	}
	return sizes
}

func (p *skylinePack) GetAreaUsedRate() float64 {
	return float64(p.usedArea) / float64(p.maxWidth*p.maxHeight)
}

func (p *skylinePack) mergeSkylines() {
	for i := 0; i < len(p.skyline)-1; i++ {
		if p.skyline[i].Y == p.skyline[i+1].Y {
			p.skyline[i].Width += p.skyline[i+1].Width
			p.skyline = slices.Delete(p.skyline, i+1, i+2)
			i--
		}
	}
}

func (p *skylinePack) testFit(index, width, height int, y *int) bool {
	x := p.skyline[index].X
	if x+width > p.maxWidth {
		return false
	}

	widthLeft := width
	i := index
	*y = p.skyline[index].Y
	for widthLeft > 0 {
		*y = max(*y, p.skyline[i].Y)
		if *y+height > p.maxHeight {
			return false
		}
		widthLeft -= p.skyline[i].Width
		i++
	}
	return true
}

func (p *skylinePack) testFitWithWaste(index, width, height int, y, wastedArea *int) bool {
	fits := p.testFit(index, width, height, y)
	if fits {
		*wastedArea = p.computeWaste(index, width, height, *y)
	}
	return fits
}

func (p *skylinePack) computeWaste(index, width, height, y int) int {
	wastedArea := 0
	rectLeft := p.skyline[index].X
	rectRight := rectLeft + width
	for index < len(p.skyline) && p.skyline[index].X < rectRight {
		if p.skyline[index].X >= rectRight || p.skyline[index].X+p.skyline[index].Width <= rectLeft {
			break
		}
		leftSide := p.skyline[index].X
		rightSide := min(rectRight, leftSide+p.skyline[index].Width)
		wastedArea += (rightSide - leftSide) * (y - p.skyline[index].Y)
		index++
	}
	return wastedArea
}

func (p *skylinePack) addWaste(index, width, height, y int) {
	// int wastedArea = 0; // unused
	rectLeft := p.skyline[index].X
	rectRight := rectLeft + width
	for index < len(p.skyline) && p.skyline[index].X < rectRight {
		if p.skyline[index].X >= rectRight || p.skyline[index].X+p.skyline[index].Width <= rectLeft {
			break
		}
		leftSide := p.skyline[index].X
		rightSide := min(rectRight, leftSide+p.skyline[index].Width)
		var waste Rect2D
		waste.X = leftSide
		waste.Y = p.skyline[index].Y
		waste.Width = rightSide - leftSide
		waste.Height = y - p.skyline[index].Y
		p.wasteMap.freeRects = append(p.wasteMap.freeRects, waste)
		index++
	}
}

func (p *skylinePack) addLevel(index int, rect *Rect2D) {
	// First track all wasted areas and mark them into the waste map if we're using one.
	if p.wasteMap != nil {
		p.addWaste(index, rect.Width, rect.Height, rect.Y)
	}

	var newNode skylineNode
	newNode.X = rect.X
	newNode.Y = rect.Y + rect.Height
	newNode.Width = rect.Width
	p.skyline = slices.Insert(p.skyline, index, newNode)

	for i := index + 1; i < len(p.skyline); i++ {
		if p.skyline[i].X < p.skyline[i-1].X+p.skyline[i-1].Width {
			shrink := p.skyline[i-1].X + p.skyline[i-1].Width - p.skyline[i].X
			p.skyline[i].X += shrink
			p.skyline[i].Width -= shrink

			if p.skyline[i].Width <= 0 {
				p.skyline = slices.Delete(p.skyline, i, i+1)
				i--
			} else {
				break
			}
		} else {
			break
		}
	}
	p.mergeSkylines()
}

func (p *skylinePack) findBottomLeft(width, height int, bestHeight, bestWidth, bestIndex *int) Rect2D {
	*bestHeight = math.MaxInt
	*bestIndex = -1
	// Used to break ties if there are nodes at the same level. Then pick the narrowest one.
	*bestWidth = math.MaxInt

	var newNode Rect2D
	for i := 0; i < len(p.skyline); i++ {
		var y int
		if p.testFit(i, width, height, &y) {
			if y+height < *bestHeight || (y+height == *bestHeight && p.skyline[i].Width < *bestWidth) {
				*bestHeight = y + height
				*bestIndex = i
				*bestWidth = p.skyline[i].Width
				newNode.X = p.skyline[i].X
				newNode.Y = y
				newNode.Width = width
				newNode.Height = height
				if newNode.IsRotated {
					newNode.RotatedCount++
				}
				newNode.IsRotated = false
			}
		}
		if p.allowRotate && p.testFit(i, height, width, &y) {
			if y+width < *bestHeight || (y+width == *bestHeight && p.skyline[i].Width < *bestWidth) {
				*bestHeight = y + width
				*bestIndex = i
				*bestWidth = p.skyline[i].Width
				newNode.X = p.skyline[i].X
				newNode.Y = y
				newNode.Width = height
				newNode.Height = width
				if !newNode.IsRotated {
					newNode.RotatedCount++
				}
				newNode.IsRotated = true
			}
		}
	}

	return newNode
}

func (p *skylinePack) findMinWaste(width, height int, bestHeight, bestWastedArea, bestIndex *int) Rect2D {
	*bestHeight = math.MaxInt
	*bestWastedArea = math.MaxInt
	*bestIndex = -1
	var newNode Rect2D

	for i := 0; i < len(p.skyline); i++ {
		var y int
		var wasted int

		if p.testFitWithWaste(i, width, height, &y, &wasted) {
			if wasted < *bestWastedArea || (wasted == *bestWastedArea && y+height < *bestHeight) {
				*bestHeight = y + height
				*bestWastedArea = wasted
				*bestIndex = i
				newNode.X = p.skyline[i].X
				newNode.Y = y
				newNode.Width = width
				newNode.Height = height
				if newNode.IsRotated {
					newNode.RotatedCount++
				}
				newNode.IsRotated = false
			}
		}

		if p.allowRotate && p.testFitWithWaste(i, height, width, &y, &wasted) {
			if wasted < *bestWastedArea || (wasted == *bestWastedArea && y+width < *bestHeight) {
				*bestHeight = y + width
				*bestWastedArea = wasted
				*bestIndex = i
				newNode.X = p.skyline[i].X
				newNode.Y = y
				newNode.Width = height
				newNode.Height = width
				if !newNode.IsRotated {
					newNode.RotatedCount++
				}
				newNode.IsRotated = true
			}
		}
	}

	return newNode
}
