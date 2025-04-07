package rectpack

// packAlgorithm 是一个包装算法的接口
type packAlgorithm interface {
	// 重置包装器到初始状态，设置最大宽高。
	// 如果宽度或高度小于1，会引发panic。
	Reset(width, height int)

	// 计算使用率，返回值在0.0（空）到1.0（完美利用）之间。
	Used() float64

	// 插入新矩形，指定矩形间的间距。
	// 返回无法包装的尺寸。
	Insert(padding int, sizes ...Size) []Size

	// 返回已包装的矩形列表。
	Rects() []Rect

	// 设置是否允许旋转矩形以优化放置。
	// 默认：false
	AllowRotate(enabled bool)

	// 返回算法可包装的最大尺寸。
	MaxSize() Size

	// 返回已使用的总面积。
	UsedArea() int
}

// algorithmBase 是一个包装算法的基础实现
type algorithmBase struct {
	packed      []Rect // 已包装的矩形
	maxWidth    int    // 包装器的最大宽度
	maxHeight   int    // 包装器的最大高度
	usedArea    int    // 已使用的面积
	allowRotate bool   // 是否允许旋转矩形
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

// Used 返回当前包装器的使用率，值在 0.0 到 1.0 之间，表示包装器的空间利用程度。
func (p *algorithmBase) Used() float64 {
	return float64(p.usedArea) / float64(p.maxWidth*p.maxHeight)
}

// Rects 返回已包装的矩形列表。
func (p *algorithmBase) Rects() []Rect {
	return p.packed
}

// AllowRotate 设置是否允许旋转矩形以优化放置。
func (p *algorithmBase) AllowRotate(enabled bool) {
	p.allowRotate = enabled
}

// MaxSize 返回包装器的最大尺寸。
func (p *algorithmBase) MaxSize() Size {
	return NewSize(p.maxWidth, p.maxHeight)
}

// UsedArea 返回已使用的总面积
func (p *algorithmBase) UsedArea() int {
	return p.usedArea
}
