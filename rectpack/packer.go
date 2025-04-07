package rectpack

import (
	"errors"
	"fmt"
	"slices"
)

// DefaultSize 定义了矩形包装器的默认最大宽度/高度值
// 基于现代GPU的最大纹理尺寸。如果这个库不是用于创建纹理图集，
// 那么这个值除了提供一个合理的起点外没有特殊意义。
const DefaultSize = 4096

// Packer 包含2D矩形包装器的状态
type Packer struct {
	// unpacked 包含尚未包装或无法包装的尺寸
	unpacked []Size

	// algo 是实现具体包装算法的实例
	algo packAlgorithm

	// sortFunc 定义在排序时用于比较尺寸大小的函数
	//
	// 默认值：SortArea
	sortFunc SortFunc

	// Padding 定义矩形周围预留的空隙大小。值为0或负数
	// 表示矩形将被紧密排列
	//
	// 默认值：0
	Padding int

	// sortRev 表示是否启用反向排序
	//
	// 默认值：false
	sortRev bool

	// Online 表示矩形是否应该在插入时立即包装(在线模式)，
	// 或者只是收集起来等待后续打包(离线模式)
	//
	// 在线/离线模式有以下权衡：
	//
	// * 在线包装更快，因为不需要排序或与其他矩形比较，
	//   但会导致优化结果较差
	// * 离线包装可能慢得多，但允许算法通过预先知道所有尺寸
	//   并进行高效排序来实现最佳效果
	//
	// 除非需要实时包装和使用结果，否则建议使用离线模式(默认)。
	// 对于创建纹理图集的任务，花费额外时间以最有效的方式准备图集是非常值得的
	//
	// 默认值：false
	Online bool
}

// Size 计算当前包装区域的尺寸，返回包含所有已包装矩形所需的最小尺寸
func (p *Packer) Size() Size {
	var size Size
	for _, rect := range p.algo.Rects() {
		size.Width = max(size.Width, rect.Right()+p.Padding)
		size.Height = max(size.Height, rect.Bottom()+p.Padding)
	}
	return size
}

// Insert 向包装器中插入多个尺寸
// 在线模式下会立即尝试包装，离线模式下只是暂存尺寸
func (p *Packer) Insert(sizes ...Size) []Size {
	// 如果启用了在线打包（Online 模式）
	if p.Online {
		// 调用具体算法的 Insert 方法，传入 Padding 和尺寸列表，返回插入结果
		return p.algo.Insert(p.Padding, sizes...)
	}
	// 否则，将尺寸追加到未打包的列表中（用于离线打包）
	p.unpacked = append(p.unpacked, sizes...)
	// 返回当前未打包的尺寸列表
	return p.unpacked
}

// InsertSize 向包装器中插入指定ID和尺寸的矩形
// 返回是否插入/包装成功
func (p *Packer) InsertSize(id, width, height int) bool {
	result := p.Insert(NewSizeID(id, width, height))
	if p.Online && len(result) != 0 {
		return false
	}
	return true
}

// Sorter 设置用于packing的排序函数和排序顺序
// 参数:
//
//	compare - 用于比较两个尺寸大小的函数
//	reverse - 是否启用反向排序
//
// 默认比较函数为 SortArea
func (p *Packer) Sorter(compare SortFunc, reverse bool) {
	p.sortFunc = compare
	p.sortRev = reverse
}

// Rects 获取所有已成功包装的矩形
// 返回:
//
//	已包装矩形的切片(由内部管理，如需修改请复制)
func (p *Packer) Rects() []Rect {
	return p.algo.Rects()
}

// Unpacked 获取所有暂存但未包装的矩形尺寸
// 返回:
//
//	未包装尺寸的切片(由内部管理，如需修改请复制)
func (p *Packer) Unpacked() []Size {
	return p.unpacked
}

// Used 计算当前空间利用率
// 参数:
//
//	current - true:计算当前区域使用率 false:计算最大可能区域使用率
//
// 返回:
//
//	空间利用率(0.0-1.0)
func (p *Packer) Used(current bool) float64 {
	if current {
		size := p.Size()
		return float64(p.algo.UsedArea()) / float64(size.Width*size.Height)
	}
	return p.algo.Used()
}

// Map 创建矩形ID到矩形对象的映射
// 返回:
//
//	映射表(map[int]Rect)
func (p *Packer) Map() map[int]Rect {
	rects := p.algo.Rects()
	mapping := make(map[int]Rect, len(rects))
	for _, rect := range rects {
		mapping[rect.ID] = rect
	}
	return mapping
}

// Clear 重置包装器状态(保留配置)
// 清除所有已包装和暂存的矩形
func (p *Packer) Clear() {
	size := p.algo.MaxSize()
	p.algo.Reset(size.Width, size.Height)
	p.unpacked = p.unpacked[:0]
}

// Pack 尝试打包所有暂存的矩形
// 返回:
//
//	true: 全部打包成功 false: 部分失败(可通过Unpacked获取失败尺寸)
func (p *Packer) Pack() bool {
	if len(p.unpacked) == 0 {
		return true
	}
	if p.sortFunc != nil {
		if p.sortRev {
			slices.SortFunc(p.unpacked, func(a, b Size) int {
				return p.sortFunc(b, a)
			})
		} else {
			slices.SortFunc(p.unpacked, p.sortFunc)
		}
	} else if p.sortRev {
		slices.Reverse(p.unpacked)
	}
	failed := p.algo.Insert(p.Padding, p.unpacked...)
	if len(failed) == 0 {
		p.unpacked = p.unpacked[:0]
		return true
	}
	p.unpacked = failed
	return false
}

// RepackAll 重新打包所有矩形(先清除再打包)
// 适用场景:
//  1. 多次打包后优化空间利用率
//  2. 修改配置后重新应用
//
// 返回:
//
//	同Pack()方法
func (p *Packer) RepackAll() bool {
	// 获取当前已打包矩形的尺寸
	rects := p.algo.Rects()
	newUnpacked := make([]Size, 0, len(rects)) // 预分配容量，避免多次扩容
	for _, rect := range rects {
		newUnpacked = append(newUnpacked, rect.Size)
	}
	// 重置打包算法状态
	size := p.Size()
	p.algo.Reset(size.Width, size.Height)
	// 替换 unpacked 队列
	p.unpacked = newUnpacked
	// 执行重新打包
	return p.Pack()
}

// AllowRotate 设置是否允许矩形旋转以优化布局
// 参数:
//
//	enabled - true:允许旋转 false:禁止旋转
//
// 默认值: false
func (p *Packer) AllowRotate(enabled bool) {
	p.algo.AllowRotate(enabled)
}

// NewPacker 创建并初始化一个新的矩形包装器
// 参数:
//
//	maxWidth - 包装区域的最大宽度(必须大于0)
//	maxHeight - 包装区域的最大高度(必须大于0)
//	heuristic - 包装算法和方法组合
//
// 返回:
//
//	*Packer - 初始化成功的包装器实例
//	error - 如果参数无效或算法不支持则返回错误
//
// 注意:
//
//	宽度或高度小于等于0会导致返回错误
func NewPacker(maxWidth, maxHeight int, heuristic Heuristic) (*Packer, error) {
	if maxWidth <= 0 || maxHeight <= 0 {
		return nil, fmt.Errorf("width and height must be greater than 0 (given %vx%x)", maxWidth, maxHeight)
	}
	p := &Packer{
		sortFunc: SortArea,
	}
	switch heuristic & typeMask {
	case MaxRects:
		p.algo = newMaxRects(maxWidth, maxHeight, heuristic)
	case Skyline:
		p.algo = newSkyline(maxWidth, maxHeight, heuristic)
	case Guillotine:
		p.algo = newGuillotine(maxWidth, maxHeight, heuristic)
	default:
		return nil, errors.New("heuristics specify an invalid argorithm")
	}
	return p, nil
}

// NewDefaultPacker 创建使用默认配置的包装器
// 默认配置:
//   - 最大尺寸: DefaultSize (4096x4096)
//   - 算法: SkylineBLF (Skyline算法+最佳长边适应启发式)
//
// 返回:
//
//	*Packer - 初始化成功的包装器实例
func NewDefaultPacker() *Packer {
	packer, _ := NewPacker(DefaultSize, DefaultSize, SkylineBLF)
	return packer
}
