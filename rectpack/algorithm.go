package rectpack

// packAlgorithm 是一个包装算法的接口
type packAlgorithm interface {
	// 重置包装器到初始状态，设置最大宽高。
	// 如果宽度或高度小于1，会引发panic。
	Reset(width, height int)
	// 计算使用率，返回值在0.0（空）到1.0（完美利用）之间。
	GetAreaUsedRate() float64
	// 插入新矩形，指定矩形间的间距。
	// 返回无法包装的尺寸。
	Insert(padding int, sizes ...Size2D) []Size2D
	// 返回已包装的矩形列表。
	GetPackedRects() []Rect2D
	// 设置是否允许旋转矩形以优化放置。
	// 默认：false
	AllowRotate(enabled bool)
	// 返回算法可包装的最大尺寸。
	MaxSize() Size2D
	// 返回已使用的总面积。
	GetUsedArea() int

	GetIdMapRotated() map[int]bool
}

// algorithmBase 是一个包装算法的基础实现
type algorithmBase struct {
	packed       []Rect2D // 已包装的矩形
	maxWidth     int      // 包装器的最大宽度
	maxHeight    int      // 包装器的最大高度
	usedArea     int      // 已使用的面积
	allowRotate  bool     // 是否允许旋转矩形
	idMapRotated map[int]bool
}

// Reset 重置包装器的状态，设置新的最大宽度和最大高度，清空已包装矩形。
//
//	width - 包装器的新最大宽度
//	height - 包装器的新最大高度
func (p *algorithmBase) Reset(width, height int) {
	p.maxWidth = width
	p.maxHeight = height
	p.usedArea = 0
	p.packed = p.packed[:0]
}

// GetAreaUsedRate 返回当前包装器的使用率，值在 0.0 到 1.0 之间，表示包装器的空间利用程度。
func (p *algorithmBase) GetAreaUsedRate() float64 {
	return float64(p.usedArea) / float64(p.maxWidth*p.maxHeight)
}

// GetPackedRects 返回已包装的矩形列表。
func (p *algorithmBase) GetPackedRects() []Rect2D {
	return p.packed
}

// AllowRotate 设置是否允许旋转矩形以优化放置。
func (p *algorithmBase) AllowRotate(enabled bool) {
	p.allowRotate = enabled
}

// MaxSize 返回包装器的最大尺寸。
func (p *algorithmBase) MaxSize() Size2D {
	return NewSize2D(p.maxWidth, p.maxHeight)
}

// GetUsedArea 返回已使用的总面积
func (p *algorithmBase) GetUsedArea() int {
	return p.usedArea
}

// 插入新矩形，指定矩形间的间距,简单按顺序放置
// 返回无法包装的尺寸。
func (p *algorithmBase) Insert(padding int, sizes ...Size2D) []Size2D {
	var unpacked []Size2D
	// 当前放置位置的坐标
	x, y := 0, 0
	// 当前行的最大高度
	rowHeight := 0
	for _, size := range sizes {
		width, height := size.Width, size.Height
		// 检查是否需要换行
		if x+width+padding > p.maxWidth {
			// 换到下一行
			x = 0
			y += rowHeight + padding
			rowHeight = 0
		}
		// 检查是否超出高度限制
		if y+height+padding > p.maxHeight {
			// 无法放置，添加到未包装列表
			unpacked = append(unpacked, size)
			continue
		}
		// 放置矩形
		rect := NewRect(x, y, width, height)
		rect.ID = size.ID
		p.packed = append(p.packed, rect)
		p.usedArea += width * height
		// 更新当前位置和行高
		x += width + padding
		if height > rowHeight {
			rowHeight = height
		}
	}
	return unpacked
}

func (p *algorithmBase) GetIdMapRotated() map[int]bool {
	return p.idMapRotated
}
