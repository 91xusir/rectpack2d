package rectpack

import "fmt"

// Point2D 描述了二维空间中的一个位置。
type Point2D struct {
	X int
	Y int
}

// NewPoint 初始化一个具有指定坐标的新点。
func NewPoint(x, y int) Point2D {
	return Point2D{X: x, Y: y}
}

// Eq 判断接收者和另一个点是否具有相同的值。
func (p *Point2D) Eq(point Point2D) bool {
	return p.X == point.X && p.Y == point.Y
}

// String 返回点的字符串表示形式。
func (p *Point2D) String() string {
	return fmt.Sprintf("[%v, %v]", p.X, p.Y)
}

// Move 将接收者的位置移动到指定的绝对坐标。
func (p *Point2D) Move(x, y int) {
	p.X = x
	p.Y = y
}

// Offset 将接收者的位置按指定的相对量移动。
func (p *Point2D) Offset(x, y int) {
	p.X += x
	p.Y += y
}

// Size2D 描述了二维空间中实体的尺寸。
type Size2D struct {
	// Width 是在水平 x 轴上的尺寸。
	Width int
	// Height 是在垂直 y 轴上的尺寸。
	Height int
	// ID 是用户定义的标识符，用于区分此实例与其他实例。
	ID int
}

// NewSize2D 创建具有指定尺寸的新尺寸对象。
func NewSize2D(width, height int) Size2D {
	return Size2D{Width: width, Height: height}
}

// NewSize2DByID 创建具有指定尺寸和唯一标识符的新尺寸对象。
func NewSize2DByID(id, width, height int) Size2D {
	return Size2D{ID: id, Width: width, Height: height}
}

// Eq 判断接收者和另一个尺寸是否具有相同的值。ID 字段被忽略。
func (sz *Size2D) Eq(size Size2D) bool {
	return sz.Width == size.Width && sz.Height == size.Height
}

// ToString 返回尺寸的字符串表示形式。
func (sz *Size2D) ToString() string {
	return fmt.Sprintf("[%v, %v]", sz.Width, sz.Height)
}

// Area 返回总面积（宽度 * 高度）。
func (sz *Size2D) Area() int {
	return sz.Width * sz.Height
}

// Perimeter 返回所有边的总长度。
func (sz *Size2D) Perimeter() int {
	return (sz.Width + sz.Height) << 1
}

// MaxSide 返回较大边的值。
func (sz *Size2D) MaxSide() int {
	return max(sz.Width, sz.Height)
}

// MinSide 返回较小边的值。
func (sz *Size2D) MinSide() int {
	return min(sz.Width, sz.Height)
}

// Ratio 计算宽度与高度之间的比率。
func (sz *Size2D) Ratio() float64 {
	return float64(sz.Width) / float64(sz.Height)
}

// Rect2D 描述了二维空间中的一个位置（左上角）和尺寸。
type Rect2D struct {
	// Point2D 表示矩形的左上角坐标。
	Point2D
	// Size2D 表示矩形的宽度和高度。
	Size2D

	IsRotated bool

	RotatedCount int
}

// NewRect 初始化一个使用指定点和尺寸值的新矩形。
func NewRect(x, y, w, h int) Rect2D {
	return Rect2D{
		Point2D: Point2D{X: x, Y: y},
		Size2D:  Size2D{Width: w, Height: h},
	}
}

// NewRectLTRB 初始化一个使用指定左/上/右/下值的新矩形。
func NewRectLTRB(l, t, r, b int) Rect2D {
	return Rect2D{
		Point2D: Point2D{X: l, Y: t},
		Size2D:  Size2D{Width: r - l, Height: b - t},
	}
}

// Eq 比较两个矩形以确定位置和尺寸是否相等。
func (r *Rect2D) Eq(rect Rect2D) bool {
	return r.Point2D.Eq(rect.Point2D) && r.Size2D.Eq(rect.Size2D)
}

// String 返回描述矩形的字符串。
func (r *Rect2D) String() string {
	return fmt.Sprintf("[%v, %v, %v, %v]", r.X, r.Y, r.Width, r.Height)
}

// Left 返回矩形左边缘在 x 轴上的坐标。
func (r *Rect2D) Left() int {
	return r.X
}

// Top 返回矩形上边缘在 y 轴上的坐标。
func (r *Rect2D) Top() int {
	return r.Y
}

// Right 返回矩形右边缘在 x 轴上的坐标。
func (r *Rect2D) Right() int {
	return r.X + r.Width
}

// Bottom 返回矩形下边缘在 y 轴上的坐标。
func (r *Rect2D) Bottom() int {
	return r.Y + r.Height
}

// TopLeft 返回表示矩形左上角的点。
func (r *Rect2D) TopLeft() Point2D {
	return Point2D{X: r.Left(), Y: r.Top()}
}

// TopRight 返回表示矩形右上角的点。
func (r *Rect2D) TopRight() Point2D {
	return Point2D{X: r.Right(), Y: r.Top()}
}

// BottomLeft 返回表示矩形左下角的点。
func (r *Rect2D) BottomLeft() Point2D {
	return Point2D{X: r.Left(), Y: r.Bottom()}
}

// BottomRight 返回表示矩形右下角的点。
func (r *Rect2D) BottomRight() Point2D {
	return Point2D{X: r.Right(), Y: r.Bottom()}
}

// Center 返回表示矩形中心的点。对于边长为二的幂的矩形，坐标向下取整。
func (r *Rect2D) Center() Point2D {
	return Point2D{X: r.X + (r.Width >> 1), Y: r.Y + (r.Height >> 1)}
}

// ContainsRect 测试指定的矩形是否包含在当前接收者的边界内。
func (r *Rect2D) ContainsRect(rect Rect2D) bool {
	return r.X <= rect.X &&
		rect.X+rect.Width <= r.X+r.Width &&
		r.Y <= rect.Y &&
		rect.Y+rect.Height <= r.Y+r.Height
}

// Contains 测试指定的坐标是否在接收者的边界内。
func (r *Rect2D) Contains(x, y int) bool {
	return r.X <= x && x < r.X+r.Width && r.Y <= y && y < r.Y+r.Height
}

// IsEmpty 测试矩形的宽度或高度是否小于1。
func (r *Rect2D) IsEmpty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Inflate 将矩形的每个边缘从中心向外推指定的相对量。
func (r *Rect2D) Inflate(width, height int) {
	r.X -= width
	r.Y -= height
	r.Width += (width << 1)
	r.Height += (height << 1)
}

// Intersects 测试接收者是否与指定的矩形有任何重叠。
func (r *Rect2D) Intersects(rect Rect2D) bool {
	return rect.X < r.X+r.Width &&
		r.X < rect.X+rect.Width &&
		rect.Y < r.Y+r.Height &&
		r.Y < rect.Y+rect.Height
}

// Intersect 返回一个仅表示此矩形与另一个矩形重叠区域的矩形，
// 如果没有重叠，则返回一个空矩形。
func (r *Rect2D) Intersect(rect Rect2D) (result Rect2D) {
	x1 := max(r.X, rect.X)
	x2 := min(r.X+r.Width, rect.X+rect.Width)
	y1 := max(r.Y, rect.Y)
	y2 := min(r.Y+r.Height, rect.Y+rect.Height)
	if x2 >= x1 && y2 >= y1 {
		result.Point2D = Point2D{X: x1, Y: y1}
		result.Size2D = Size2D{Width: x2 - x1, Height: y2 - y1}
	}
	return
}

// Union 返回一个包含目标和自己的最小矩形
func (r *Rect2D) Union(rect Rect2D) Rect2D {
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
func padSize(size *Size2D, padding int) {
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
func unpadRect(rect *Rect2D, padding int) {
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
