package main

import (
	"container/heap"
	"fmt"
	"math"
)

// Box 代表盒子结构体

type Box struct {
	// The width of the boundary

	W float64
	// The height of the boundary

	H float64
	// Rectangle list

	ReqPackRectList []*Rect
	// Whether to allow rectangle rotation

	IsRotateEnable bool
}

// Rect represents a rectangular structure

type Rect struct {
	id uint
	w  float64
	h  float64
}

func NewRect(id uint, w float64, h float64) *Rect {
	return &Rect{
		id: id,
		w:  w,
		h:  h,
	}
}

func Copy(r *Rect) *Rect {
	return NewRect(r.id, r.w, r.h)
}

func CopySlice(rs []*Rect) []*Rect {
	newItems := make([]*Rect, len(rs))
	for i, item := range rs {
		newItems[i] = Copy(item)
	}
	return newItems
}

// A skyline structure

type SkyLine struct {
	x   float64
	y   float64
	len float64
}

func (s *SkyLine) String() string {
	return fmt.Sprintf("SkyLine{x=%f, y=%f, len=%f}", s.x, s.y, s.len)
}

func NewSkyLine(x, y, len float64) *SkyLine {
	return &SkyLine{
		x:   x,
		y:   y,
		len: len,
	}
}

// The smallest pile of skyline

type SkyLineHeap []*SkyLine

func (h SkyLineHeap) Len() int { return len(h) }

func (h SkyLineHeap) Less(i, j int) bool {
	if h[i].y == h[j].y {
		return h[i].x < h[j].x
	}
	return h[i].y < h[j].y
}
func (h SkyLineHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *SkyLineHeap) Push(x any) {
	*h = append(*h, x.(*SkyLine))
}

func (h *SkyLineHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// PackedRect 打包后的矩形对象，包含坐标和尺寸信息，以及是否旋转的标志
// 注意：x和y表示左上角的坐标，w和h表示矩形的宽和高

type PackedRect struct {
	id        uint
	x         float64
	y         float64
	w         float64
	h         float64
	isRotated bool // 是否旋转

}

func NewPackedRect(id uint, x, y, w, h float64, isRotate bool) *PackedRect {
	return &PackedRect{
		id:        id,
		isRotated: isRotate,
		x:         x,
		y:         y,
		w:         w,
		h:         h,
	}
}

// Packaging results

type PackResult struct {
	packedRectList []*PackedRect // Packed rectangle list

	totalS float64 // Total area

	rate float64 // Fill rate

}

func NewSolution(packedRectList []*PackedRect, totalS float64, rate float64) *PackResult {
	return &PackResult{
		packedRectList: packedRectList,
		totalS:         totalS,
		rate:           rate,
	}
}

// SkyLinePacking 主结构体

type SkyLinePacking struct {
	w float64 // Total width

	h float64 // Total height

	reqPackRectList []*Rect // Rectangle to be packaged

	isRotateEnable bool // Whether to allow rotation

	skyLineQueue SkyLineHeap // The smallest pile of skyline

}

// Constructor

func NewSkyLinePacking(isRotateEnable bool, w, h float64, rects []*Rect) *SkyLinePacking {
	sp := &SkyLinePacking{
		w:               w,
		h:               h,
		reqPackRectList: rects,
		isRotateEnable:  isRotateEnable,
		skyLineQueue:    SkyLineHeap{},
	}
	heap.Init(&sp.skyLineQueue)
	sp.skyLineQueue = append(sp.skyLineQueue, &SkyLine{x: 0, y: 0, len: w})
	return sp
}

func (sp *SkyLinePacking) Pack() (*PackResult, error) {
	// Used to record the total area of ​​the rectangle that has been placed

	totalS := 0.0
	// Used to store already placed rectangles

	packedRectList := make([]*PackedRect, 0, len(sp.reqPackRectList))
	// Record the rectangle that has been placed

	used := make([]bool, len(sp.reqPackRectList))
	for sp.skyLineQueue.Len() != 0 && len(packedRectList) < len(sp.reqPackRectList) {
		// Get the current lowest and leftmost skyline (retrieve the first element of the queue)

		skyLine := heap.Pop(&sp.skyLineQueue).(*SkyLine)
		// Initialize hl and hr

		hl := sp.h - skyLine.y
		hr := sp.h - skyLine.y
		count := 0

		// Iterate through the skyline queue in sequence, and get hl and hr according to the skyline and skyline queues

		for _, line := range sp.skyLineQueue {
			if comparefloat64(line.x+line.len, skyLine.x) == 0 {
				hl = line.y - skyLine.y
				count++
			} else if comparefloat64(line.x, skyLine.x+skyLine.len) == 0 {
				hr = line.y - skyLine.y
				count++
			}
			if count == 2 {
				break
			}
		}
		// Record the index of the maximum score rectangle, the maximum score

		maxRectIndex, maxScore := -1, -1
		// Record whether the rectangle with the maximum score rotates

		isRotate := false
		// Iterate through each rectangle and select the largest rating rectangle for placement

		for i := range sp.reqPackRectList {
			// The rectangle has not been placed before proceeding

			if !used[i] {
				// No rotation

				score := sp.score(sp.reqPackRectList[i].w, sp.reqPackRectList[i].h, skyLine, hl, hr)
				if score > maxScore {
					maxScore = score
					maxRectIndex = i
					isRotate = false
				}
				// Rotational situation

				if sp.isRotateEnable {
					rotateScore := sp.score(sp.reqPackRectList[i].h, sp.reqPackRectList[i].w, skyLine, hl, hr)
					if rotateScore > maxScore {
						maxScore = rotateScore
						maxRectIndex = i
						isRotate = true
					}
				}
			}

		}
		// If the current maximum score is greater than or equal to 0, it means that there is a rectangle that can be placed, and then place it according to the rules

		if maxScore >= 0 {
			// The left wall is higher than or equal to the right wall

			if hl >= hr {
				// When the score is 2, the rectangle is placed on the right side of the skyline, otherwise it is placed on the left side of the skyline.

				if maxScore == 2 {
					packedRect := sp.placeRight(sp.reqPackRectList[maxRectIndex], skyLine, isRotate)
					packedRectList = append(packedRectList, packedRect)
				} else {
					packedRect := sp.placeLeft(sp.reqPackRectList[maxRectIndex], skyLine, isRotate)
					packedRectList = append(packedRectList, packedRect)
				}

			} else {
				if maxScore == 4 || maxScore == 0 {
					packedRect := sp.placeRight(sp.reqPackRectList[maxRectIndex], skyLine, isRotate)
					packedRectList = append(packedRectList, packedRect)
				} else {
					packedRect := sp.placeLeft(sp.reqPackRectList[maxRectIndex], skyLine, isRotate)
					packedRectList = append(packedRectList, packedRect)
				}

			}
			used[maxRectIndex] = true
			totalS += sp.reqPackRectList[maxRectIndex].w * sp.reqPackRectList[maxRectIndex].h
		} else {
			sp.combineSkylines(skyLine)
		}
	}
	return &PackResult{packedRectList, totalS, totalS / (sp.w * sp.h)}, nil
}

// placeLeft 将矩形靠左放

func (sp *SkyLinePacking) placeLeft(rect *Rect, skyLine *SkyLine, isRotate bool) *PackedRect {
	var packedRect *PackedRect
	if !isRotate {
		packedRect = NewPackedRect(rect.id, skyLine.x, skyLine.y, rect.w, rect.h, isRotate)
	} else {
		packedRect = NewPackedRect(rect.id, skyLine.x, skyLine.y, rect.h, rect.w, isRotate)
	}
	sp.addSkyLineInQueue(skyLine.x, skyLine.y+packedRect.h, packedRect.w)
	sp.addSkyLineInQueue(skyLine.x+packedRect.w, skyLine.y, skyLine.len-packedRect.w)
	return packedRect
}

// placeRight Place the rectangle to the right

func (sp *SkyLinePacking) placeRight(rect *Rect, skyLine *SkyLine, isRotate bool) *PackedRect {
	var packedRect *PackedRect
	if !isRotate {
		packedRect = NewPackedRect(rect.id, skyLine.x+skyLine.len-rect.w, skyLine.y, rect.w, rect.h, isRotate)
	} else {
		packedRect = NewPackedRect(rect.id, skyLine.x+skyLine.len-rect.h, skyLine.y, rect.h, rect.w, isRotate)
	}
	sp.addSkyLineInQueue(skyLine.x, skyLine.y, skyLine.len-packedRect.w)
	sp.addSkyLineInQueue(packedRect.x, skyLine.y+packedRect.h, packedRect.w)
	return packedRect
}

// addSkyLineInQueue 将指定属性的天际线加入天际线队列

func (sp *SkyLinePacking) addSkyLineInQueue(x, y, len float64) {
	if comparefloat64(len, 0.0) == 1 {
		skyLine := &SkyLine{x: x, y: y, len: len}
		heap.Push(&sp.skyLineQueue, skyLine)
	}
}

func (sp *SkyLinePacking) combineSkylines(skyLine *SkyLine) {
	b := false
	for i, line := range sp.skyLineQueue {
		if comparefloat64(skyLine.y, line.y) != 1 {
			if comparefloat64(skyLine.x, line.x+line.len) == 0 {
				heap.Remove(&sp.skyLineQueue, i)
				b = true
				skyLine.x = line.x
				skyLine.y = line.y
				skyLine.len = line.len + skyLine.len
				break
			}
			if comparefloat64(skyLine.x+skyLine.len, line.x) == 0 {
				heap.Remove(&sp.skyLineQueue, i)
				b = true
				skyLine.y = line.y
				skyLine.len = line.len + skyLine.len
				break
			}
		}
	}
	if b {
		heap.Push(&sp.skyLineQueue, skyLine)
	}
}

// score 对矩形进行评分，如果评分为 -1 ，则说明该矩形不能放置在该天际线上

func (sp *SkyLinePacking) score(w, h float64, skyLine *SkyLine, hl, hr float64) int {
	// The current skyline length is smaller than the current rectangle width and cannot be put down

	if comparefloat64(skyLine.len, w) == -1 {
		return -1
	}
	// If it exceeds the upper bound, it cannot be released

	if comparefloat64(skyLine.y+h, sp.h) == 1 {
		return -1
	}
	score := -1
	// The left wall is higher than or equal to the right wall

	if hl >= hr {
		if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hl) == 0 {
			score = 7
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hr) == 0 {
			score = 6
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hl) == 1 {
			score = 5
		} else if comparefloat64(w, skyLine.len) == -1 && comparefloat64(h, hl) == 0 {
			score = 4
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hl) == -1 && comparefloat64(h, hr) == 1 {
			score = 3
		} else if comparefloat64(w, skyLine.len) == -1 && comparefloat64(h, hr) == 0 {
			score = 2
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hr) == -1 {
			score = 1
		} else if comparefloat64(w, skyLine.len) == -1 && comparefloat64(h, hl) != 0 {
			score = 0
		} else {
			panic(fmt.Sprintf("w = %f , h = %f , hl = %f , hr = %f , skyline = %+v", w, h, hl, hr, skyLine))
		}
	} else {
		if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hr) == 0 {
			score = 7
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hl) == 0 {
			score = 6
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hr) == 1 {
			score = 5
		} else if comparefloat64(w, skyLine.len) == -1 && comparefloat64(h, hr) == 0 {
			score = 4
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hr) == -1 && comparefloat64(h, hl) == 1 {
			score = 3
		} else if comparefloat64(w, skyLine.len) == -1 && comparefloat64(h, hl) == 0 {
			score = 2
		} else if comparefloat64(w, skyLine.len) == 0 && comparefloat64(h, hl) == -1 {
			score = 1
		} else if comparefloat64(w, skyLine.len) == -1 && comparefloat64(h, hr) != 0 {
			score = 0
		} else {
			panic(fmt.Sprintf("w = %f , h = %f , hl = %f , hr = %f , skyline = %+v", w, h, hl, hr, skyLine))
		}
	}
	return score
}

// Compare the size of two floating point numbers, with an accuracy of 1e 6

func comparefloat64(a, b float64) int {
	if math.Abs(a-b) < 1e-6 {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}
