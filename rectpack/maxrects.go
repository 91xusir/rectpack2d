package rectpack

import "math"

type heuristicFunc func(pack *maxRects, width, height int) (Rect2D, int, int)

type maxRects struct {
	algorithmBase
	findNode     heuristicFunc
	newLastSize  int
	newFreeRects []Rect2D
	freeRects    []Rect2D
}

func newMaxRects(width, height int, heuristic Heuristic) *maxRects {
	var p maxRects
	p.idMaptoRotateCount = make(map[int]int)
	switch heuristic & fitMask {
	case BestAreaFit:
		p.findNode = findPositionBestAreaFit
	case BottomLeft:
		p.findNode = findPositionBottomLeft
	case ContactPoint:
		p.findNode = findPositionContactPoint
	case BestLongSideFit:
		p.findNode = findPositionBestLongSideFit
	case BestShortSideFit:
		p.findNode = findPositionBestShortSideFit
	default: // BestShortSideFit
		p.findNode = findPositionBestShortSideFit
	}

	p.Reset(width, height)
	return &p
}

func (p *maxRects) Reset(width, height int) {
	p.algorithmBase.Reset(width, height)
	p.newFreeRects = p.newFreeRects[:0]
	p.freeRects = p.freeRects[:0]
	p.freeRects = append(p.freeRects, NewRect(0, 0, p.maxWidth, p.maxHeight))
}

func (p *maxRects) Insert(padding int, sizes ...Size2D) []Size2D {
	for len(sizes) > 0 {

		var bestNode Rect2D
		bestScore1 := math.MaxInt
		bestScore2 := math.MaxInt
		bestRectIndex := -1

		for i, size := range sizes {

			padSize(&size, padding)
			newNode, score1, score2 := p.scoreRect(size.Width, size.Height)
			if score1 < bestScore1 || (score1 == bestScore1 && score2 < bestScore2) {
				bestScore1 = score1
				bestScore2 = score2
				bestNode = newNode
				bestNode.ID = size.ID
				bestRectIndex = i
			}
		}

		if bestRectIndex == -1 {
			break
		}

		p.placeRect(bestNode)
		unpadRect(&bestNode, padding)
		p.idMaptoRotateCount[bestNode.ID] += bestNode.RotatedCount
		p.packed = append(p.packed, bestNode)

		last := len(sizes) - 1
		sizes[bestRectIndex] = sizes[last]
		sizes = sizes[:last]
	}
	return sizes
}

func (p *maxRects) scoreRect(width, height int) (Rect2D, int, int) {
	newNode, score1, score2 := p.findNode(p, width, height)
	if newNode.Height == 0 {
		score1 = math.MaxInt
		score2 = math.MaxInt
	}
	return newNode, score1, score2
}

func (p *maxRects) placeRect(node Rect2D) {
	for i := 0; i < len(p.freeRects); {
		if p.splitFreeNode(&p.freeRects[i], &node) {
			last := len(p.freeRects) - 1
			p.freeRects[i] = p.freeRects[last]
			p.freeRects = p.freeRects[:last]
		} else {
			i++
		}
	}
	p.pruneFreeList()
	p.usedArea += node.Area()
}

func findPositionBottomLeft(p *maxRects, width, height int) (Rect2D, int, int) {
	var bestNode Rect2D

	bestY := math.MaxInt
	bestX := math.MaxInt

	for _, freeRect := range p.freeRects {

		// Try to place the rectangle in upright (non-flipped) orientation.
		if freeRect.Width >= width && freeRect.Height >= height {
			topSideY := freeRect.Y + height
			if topSideY < bestY || (topSideY == bestY && freeRect.X < bestX) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = width
				bestNode.Height = height
				bestY = topSideY
				bestX = freeRect.X
				if bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = false
			}
		}

		if p.allowRotate && freeRect.Width >= height && freeRect.Height >= width {
			topSideY := freeRect.Y + width
			if topSideY < bestY || (topSideY == bestY && freeRect.X < bestX) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = height
				bestNode.Height = width
				bestY = topSideY
				bestX = freeRect.X
				if !bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = true
			}
		}
	}
	return bestNode, bestY, bestX
}

func findPositionBestShortSideFit(p *maxRects, width, height int) (Rect2D, int, int) {
	var bestNode Rect2D
	bestShortSideFit := math.MaxInt
	bestLongSideFit := math.MaxInt

	for _, freeRect := range p.freeRects {

		// Try to place the rectangle in upright (non-flipped) orientation.
		if freeRect.Width >= width && freeRect.Height >= height {

			leftoverHoriz := abs(freeRect.Width - width)
			leftoverVert := abs(freeRect.Height - height)
			shortSideFit := min(leftoverHoriz, leftoverVert)
			longSideFit := max(leftoverHoriz, leftoverVert)

			if shortSideFit < bestShortSideFit || (shortSideFit == bestShortSideFit && longSideFit < bestLongSideFit) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = width
				bestNode.Height = height
				bestShortSideFit = shortSideFit
				bestLongSideFit = longSideFit
				if bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = false
			}
		}

		if p.allowRotate && freeRect.Width >= height && freeRect.Height >= width {
			flippedLeftoverHoriz := abs(freeRect.Width - height)
			flippedLeftoverVert := abs(freeRect.Height - width)
			flippedShortSideFit := min(flippedLeftoverHoriz, flippedLeftoverVert)
			flippedLongSideFit := max(flippedLeftoverHoriz, flippedLeftoverVert)

			if flippedShortSideFit < bestShortSideFit || (flippedShortSideFit == bestShortSideFit && flippedLongSideFit < bestLongSideFit) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = height
				bestNode.Height = width
				bestShortSideFit = flippedShortSideFit
				bestLongSideFit = flippedLongSideFit
				if !bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = true
			}
		}
	}

	return bestNode, bestShortSideFit, bestLongSideFit
}

func findPositionBestLongSideFit(p *maxRects, width, height int) (Rect2D, int, int) {
	var bestNode Rect2D
	bestShortSideFit := math.MaxInt
	bestLongSideFit := math.MaxInt

	for _, freeRect := range p.freeRects {

		// Try to place the rectangle in upright (non-flipped) orientation.
		if freeRect.Width >= width && freeRect.Height >= height {

			leftoverHoriz := abs(freeRect.Width - width)
			leftoverVert := abs(freeRect.Height - height)
			shortSideFit := min(leftoverHoriz, leftoverVert)
			longSideFit := max(leftoverHoriz, leftoverVert)

			if longSideFit < bestLongSideFit || (longSideFit == bestLongSideFit && shortSideFit < bestShortSideFit) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = width
				bestNode.Height = height
				if bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = false
				bestShortSideFit = shortSideFit
				bestLongSideFit = longSideFit
			}
		}

		if p.allowRotate && freeRect.Width >= height && freeRect.Height >= width {
			leftoverHoriz := abs(freeRect.Width - height)
			leftoverVert := abs(freeRect.Height - width)
			shortSideFit := min(leftoverHoriz, leftoverVert)
			longSideFit := max(leftoverHoriz, leftoverVert)

			if longSideFit < bestLongSideFit || (longSideFit == bestLongSideFit && shortSideFit < bestShortSideFit) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = height
				bestNode.Height = width
				bestShortSideFit = shortSideFit
				bestLongSideFit = longSideFit
				if !bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = true
			}
		}
	}
	return bestNode, bestShortSideFit, bestLongSideFit
}

func findPositionBestAreaFit(p *maxRects, width, height int) (Rect2D, int, int) {
	var bestNode Rect2D
	bestAreaFit := math.MaxInt
	bestShortSideFit := math.MaxInt
	for _, freeRect := range p.freeRects {
		areaFit := freeRect.Width*freeRect.Height - width*height
		// Try to place the rectangle in upright (non-flipped) orientation.
		if freeRect.Width >= width && freeRect.Height >= height {
			leftoverHoriz := abs(freeRect.Width - width)
			leftoverVert := abs(freeRect.Height - height)
			shortSideFit := min(leftoverHoriz, leftoverVert)
			if areaFit < bestAreaFit || (areaFit == bestAreaFit && shortSideFit < bestShortSideFit) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = width
				bestNode.Height = height
				if bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = false
				bestShortSideFit = shortSideFit
				bestAreaFit = areaFit
			}
		}

		if p.allowRotate && freeRect.Width >= height && freeRect.Height >= width {
			leftoverHoriz := abs(freeRect.Width - height)
			leftoverVert := abs(freeRect.Height - width)
			shortSideFit := min(leftoverHoriz, leftoverVert)
			if areaFit < bestAreaFit || (areaFit == bestAreaFit && shortSideFit < bestShortSideFit) {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = height
				bestNode.Height = width
				if !bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = true
				bestShortSideFit = shortSideFit
				bestAreaFit = areaFit
			}
		}
	}
	return bestNode, bestAreaFit, bestShortSideFit
}

// Returns 0 if the two intervals i1 and i2 are disjoint, or the length of their overlap otherwise
func (p *maxRects) commonIntervalLength(i1start, i1end, i2start, i2end int) int {
	if i1end < i2start || i2end < i1start {
		return 0
	}
	return min(i1end, i2end) - max(i1start, i2start)
}

func (p *maxRects) contactPointScoreNode(x, y, width, height int) int {
	score := 0

	if x == 0 || x+width == p.maxWidth {
		score += height
	}
	if y == 0 || y+height == p.maxHeight {
		score += width
	}

	for _, used := range p.packed {

		if used.X == x+width || used.X+used.Width == x {
			score += p.commonIntervalLength(used.Y, used.Y+used.Height, y, y+height)
		}
		if used.Y == y+height || used.Y+used.Height == y {
			score += p.commonIntervalLength(used.X, used.X+used.Width, x, x+width)
		}
	}
	return score
}

func findPositionContactPoint(p *maxRects, width, height int) (Rect2D, int, int) {
	var bestNode Rect2D
	bestContactScore := -1

	for _, freeRect := range p.freeRects {
		// Try to place the rectangle in upright (non-flipped) orientation.
		if freeRect.Width >= width && freeRect.Height >= height {
			score := p.contactPointScoreNode(freeRect.X, freeRect.Y, width, height)
			if score > bestContactScore {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = width
				bestNode.Height = height
				if bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = false
				bestContactScore = score
			}
		}
		if p.allowRotate && freeRect.Width >= height && freeRect.Height >= width {
			score := p.contactPointScoreNode(freeRect.X, freeRect.Y, height, width)
			if score > bestContactScore {
				bestNode.X = freeRect.X
				bestNode.Y = freeRect.Y
				bestNode.Width = height
				bestNode.Height = width
				if !bestNode.IsRotated {
					bestNode.RotatedCount++
				}
				bestNode.IsRotated = true
				bestContactScore = score
			}
		}
	}
	return bestNode, bestContactScore, math.MaxInt
}

func (p *maxRects) insertNewFreeRectangle(newFreeRect Rect2D) {
	for i := 0; i < p.newLastSize; {
		// This new free rectangle is already accounted for?
		if p.newFreeRects[i].ContainsRect(newFreeRect) {
			return
		}

		// Does this new free rectangle obsolete a previous new free rectangle?
		if newFreeRect.ContainsRect(p.newFreeRects[i]) {
			// Remove i'th new free rectangle, but do so by retaining the order
			// of the older vs newest free rectangles that we may still be placing
			// in calling function SplitFreeNode().
			p.newLastSize--
			p.newFreeRects[i] = p.newFreeRects[p.newLastSize]

			last := len(p.newFreeRects) - 1
			p.newFreeRects[p.newLastSize] = p.newFreeRects[last]
			p.newFreeRects = p.newFreeRects[:last]
			continue
		}

		i++
	}

	p.newFreeRects = append(p.newFreeRects, newFreeRect)
}

func (p *maxRects) splitFreeNode(freeNode, usedNode *Rect2D) bool {
	if usedNode.X >= freeNode.X+freeNode.Width || usedNode.X+usedNode.Width <= freeNode.X || usedNode.Y >= freeNode.Y+freeNode.Height || usedNode.Y+usedNode.Height <= freeNode.Y {
		return false
	}

	p.newLastSize = len(p.newFreeRects)

	if usedNode.X < freeNode.X+freeNode.Width && usedNode.X+usedNode.Width > freeNode.X {
		// New node at the top side of the used node.
		if usedNode.Y > freeNode.Y && usedNode.Y < freeNode.Y+freeNode.Height {
			newNode := *freeNode
			newNode.Height = usedNode.Y - newNode.Y
			p.insertNewFreeRectangle(newNode)
		}

		// New node at the bottom side of the used node.
		if usedNode.Y+usedNode.Height < freeNode.Y+freeNode.Height {
			newNode := *freeNode
			newNode.Y = usedNode.Y + usedNode.Height
			newNode.Height = freeNode.Y + freeNode.Height - (usedNode.Y + usedNode.Height)
			p.insertNewFreeRectangle(newNode)
		}
	}

	if usedNode.Y < freeNode.Y+freeNode.Height && usedNode.Y+usedNode.Height > freeNode.Y {
		// New node at the left side of the used node.
		if usedNode.X > freeNode.X && usedNode.X < freeNode.X+freeNode.Width {
			newNode := *freeNode
			newNode.Width = usedNode.X - newNode.X
			p.insertNewFreeRectangle(newNode)
		}

		// New node at the right side of the used node.
		if usedNode.X+usedNode.Width < freeNode.X+freeNode.Width {
			newNode := *freeNode
			newNode.X = usedNode.X + usedNode.Width
			newNode.Width = freeNode.X + freeNode.Width - (usedNode.X + usedNode.Width)
			p.insertNewFreeRectangle(newNode)
		}
	}

	return true
}

func (p *maxRects) pruneFreeList() {
	// Test all newly introduced free rectangles against old free rectangles.
	for i := 0; i < len(p.freeRects); i++ {
		for j := 0; j < len(p.newFreeRects); {

			if p.freeRects[i].ContainsRect(p.newFreeRects[j]) {

				last := len(p.newFreeRects) - 1
				p.newFreeRects[j] = p.newFreeRects[last]
				p.newFreeRects = p.newFreeRects[:last]
				continue
			}
			j++
		}
	}

	// Merge new and old free rectangles to the group of old free rectangles.
	p.freeRects = append(p.freeRects, p.newFreeRects...)
	p.newFreeRects = p.newFreeRects[:0]
}
