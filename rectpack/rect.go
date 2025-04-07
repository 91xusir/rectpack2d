package rectpack

import "fmt"

// Point 描述了二维空间中的一个位置。
type Point struct {
	// X 是在水平 x 轴上的位置。
	X int `json:"x"`
	// Y 是在垂直 y 轴上的位置。
	Y int `json:"y"`
}

// NewPoint 初始化一个具有指定坐标的新点。
func NewPoint(x, y int) Point {
	return Point{X: x, Y: y}
}

// Eq 判断接收者和另一个点是否具有相同的值。
func (p *Point) Eq(point Point) bool {
	return p.X == point.X && p.Y == point.Y
}

// String 返回点的字符串表示形式。
func (p *Point) String() string {
	return fmt.Sprintf("[%v, %v]", p.X, p.Y)
}

// Move 将接收者的位置移动到指定的绝对坐标。
func (p *Point) Move(x, y int) {
	p.X = x
	p.Y = y
}

// Offset 将接收者的位置按指定的相对量移动。
func (p *Point) Offset(x, y int) {
	p.X += x
	p.Y += y
}

// Size 描述了二维空间中实体的尺寸。
type Size struct {
	// Width 是在水平 x 轴上的尺寸。
	Width int `json:"width"`
	// Height 是在垂直 y 轴上的尺寸。
	Height int `json:"height"`
	// ID 是用户定义的标识符，用于区分此实例与其他实例。
	ID int `json:"-"`
}

// NewSize 创建具有指定尺寸的新尺寸对象。
func NewSize(width, height int) Size {
	return Size{Width: width, Height: height}
}

// NewSizeID 创建具有指定尺寸和唯一标识符的新尺寸对象。
func NewSizeID(id, width, height int) Size {
	return Size{ID: id, Width: width, Height: height}
}

// Eq 判断接收者和另一个尺寸是否具有相同的值。ID 字段被忽略。
func (sz *Size) Eq(size Size) bool {
	return sz.Width == size.Width && sz.Height == size.Height
}

// String 返回尺寸的字符串表示形式。
func (sz *Size) String() string {
	return fmt.Sprintf("[%v, %v]", sz.Width, sz.Height)
}

// Area 返回总面积（宽度 * 高度）。
func (sz *Size) Area() int {
	return sz.Width * sz.Height
}

// Perimeter 返回所有边的总长度。
func (sz *Size) Perimeter() int {
	return (sz.Width + sz.Height) << 1
}

// MaxSide 返回较大边的值。
func (sz *Size) MaxSide() int {
	return max(sz.Width, sz.Height)
}

// MinSide 返回较小边的值。
func (sz *Size) MinSide() int {
	return min(sz.Width, sz.Height)
}

// Ratio 计算宽度与高度之间的比率。
func (sz *Size) Ratio() float64 {
	return float64(sz.Width) / float64(sz.Height)
}

// Rect 描述了二维空间中的一个位置（左上角）和尺寸。
type Rect struct {
	// Point 表示矩形的左上角坐标。
	Point
	// Size 表示矩形的宽度和高度。
	Size
	// Rotated 指示矩形是否已旋转。
	Rotated bool `json:"flipped,omitempty"`
}

// NewRect 初始化一个使用指定点和尺寸值的新矩形。
func NewRect(x, y, w, h int) Rect {
	return Rect{
		Point: Point{X: x, Y: y},
		Size:  Size{Width: w, Height: h},
	}
}

// NewRectLTRB 初始化一个使用指定左/上/右/下值的新矩形。
func NewRectLTRB(l, t, r, b int) Rect {
	return Rect{
		Point: Point{X: l, Y: t},                 // 左上角坐标，Y 应该是 t，而不是 r
		Size:  Size{Width: r - l, Height: b - t}, // 计算宽度和高度
	}
}

// Eq 比较两个矩形以确定位置和尺寸是否相等。
func (r *Rect) Eq(rect Rect) bool {
	return r.Point.Eq(rect.Point) && r.Size.Eq(rect.Size)
}

// String 返回描述矩形的字符串。
func (r *Rect) String() string {
	return fmt.Sprintf("[%v, %v, %v, %v]", r.X, r.Y, r.Width, r.Height)
}

// Left 返回矩形左边缘在 x 轴上的坐标。
func (r *Rect) Left() int {
	return r.X
}

// Top 返回矩形上边缘在 y 轴上的坐标。
func (r *Rect) Top() int {
	return r.Y
}

// Right 返回矩形右边缘在 x 轴上的坐标。
func (r *Rect) Right() int {
	return r.X + r.Width
}

// Bottom 返回矩形下边缘在 y 轴上的坐标。
func (r *Rect) Bottom() int {
	return r.Y + r.Height
}

// TopLeft 返回表示矩形左上角的点。
func (r *Rect) TopLeft() Point {
	return Point{X: r.Left(), Y: r.Top()}
}

// TopRight 返回表示矩形右上角的点。
func (r *Rect) TopRight() Point {
	return Point{X: r.Right(), Y: r.Top()}
}

// BottomLeft 返回表示矩形左下角的点。
func (r *Rect) BottomLeft() Point {
	return Point{X: r.Left(), Y: r.Bottom()}
}

// BottomRight 返回表示矩形右下角的点。
func (r *Rect) BottomRight() Point {
	return Point{X: r.Right(), Y: r.Bottom()}
}

// Center 返回表示矩形中心的点。对于边长为二的幂的矩形，坐标向下取整。
func (r *Rect) Center() Point {
	return Point{X: r.X + (r.Width >> 1), Y: r.Y + (r.Height >> 1)}
}

// ContainsRect 测试指定的矩形是否包含在当前接收者的边界内。
func (r *Rect) ContainsRect(rect Rect) bool {
	return r.X <= rect.X &&
		rect.X+rect.Width <= r.X+r.Width &&
		r.Y <= rect.Y &&
		rect.Y+rect.Height <= r.Y+r.Height
}

// Contains 测试指定的坐标是否在接收者的边界内。
func (r *Rect) Contains(x, y int) bool {
	return r.X <= x && x < r.X+r.Width && r.Y <= y && y < r.Y+r.Height
}

// IsEmpty 测试矩形的宽度或高度是否小于1。
func (r *Rect) IsEmpty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Inflate 将矩形的每个边缘从中心向外推指定的相对量。
func (r *Rect) Inflate(width, height int) {
	r.X -= width
	r.Y -= height
	r.Width += (width << 1)
	r.Height += (height << 1)
}

// Intersects 测试接收者是否与指定的矩形有任何重叠。
func (r *Rect) Intersects(rect Rect) bool {
	return rect.X < r.X+r.Width &&
		r.X < rect.X+rect.Width &&
		rect.Y < r.Y+r.Height &&
		r.Y < rect.Y+rect.Height
}

// Intersect 返回一个仅表示此矩形与另一个矩形重叠区域的矩形，
// 如果没有重叠，则返回一个空矩形。
func (r *Rect) Intersect(rect Rect) (result Rect) {
	x1 := max(r.X, rect.X)
	x2 := min(r.X+r.Width, rect.X+rect.Width)
	y1 := max(r.Y, rect.Y)
	y2 := min(r.Y+r.Height, rect.Y+rect.Height)
	if x2 >= x1 && y2 >= y1 {
		result.Point = Point{X: x1, Y: y1}
		result.Size = Size{Width: x2 - x1, Height: y2 - y1}
	}
	return
}

// Union 返回一个包含目标和自己的最小矩形
func (r *Rect) Union(rect Rect) Rect {
	x1 := min(r.X, rect.X)
	x2 := max(r.X+r.Width, rect.X+rect.Width)
	y1 := min(r.Y, rect.Y)
	y2 := max(r.Y+r.Height, rect.Y+rect.Height)
	return NewRect(x1, y1, x2-x1, y2-y1)
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x >= 0 {
		return x
	}
	return -x
}

// padSize 在给定的尺寸上加上指定的间距
//
//	size - 要修改的尺寸指针
//	padding - 要添加的间距大小
func padSize(size *Size, padding int) {
	if padding <= 0 {
		return
	}
	size.Width += padding
	size.Height += padding
}

// unpadRect 从矩形中移除内边距
//
//	rect - 要修改的矩形指针
//	padding - 要移除的内边距大小
func unpadRect(rect *Rect, padding int) {
	if padding <= 0 {
		return
	}
	if rect.X == 0 {
		rect.X += padding
		rect.Width -= padding * 2
	} else {
		rect.Width -= padding
	}
	if rect.Y == 0 {
		rect.Y += padding
		rect.Height -= padding * 2
	} else {
		rect.Height -= padding
	}
}
