package rectpack

import (
	"fmt"
	"math"
	"slices"
)

const DefaultSize = 4096

type Packer struct {
	unpackedSize2Ds []Size2D
	algo            packAlgorithm
	sortFunc        SortFunc
	padding         int
	sortRev         bool
	Online          bool
}

func (p *Packer) MaxSize() Size2D {
	return p.algo.MaxSize()
}

func (p *Packer) GetIdMapRotated() map[int]bool {
	return p.algo.GetIdMapRotated()
}

// MinSize 包含所有已包装矩形所需的最小尺寸
func (p *Packer) MinSize() Size2D {
	var size Size2D
	for _, rect := range p.algo.GetPackedRects() {
		size.Width = max(size.Width, rect.Right()+p.padding)
		size.Height = max(size.Height, rect.Bottom()+p.padding)
	}
	return size
}

// Insert 向包装器中插入多个尺寸
// 在线模式下会立即尝试包装，离线模式下只是暂存尺寸
func (p *Packer) Insert(sizes ...Size2D) []Size2D {
	// 如果启用了在线打包（Online 模式）
	if p.Online {
		// 调用具体算法的 Insert 方法，传入 Padding 和尺寸列表，返回插入结果
		return p.algo.Insert(p.padding, sizes...)
	}
	// 否则，将尺寸追加到未打包的列表中（用于离线打包）
	p.unpackedSize2Ds = append(p.unpackedSize2Ds, sizes...)
	// 返回当前未打包的尺寸列表
	return p.unpackedSize2Ds
}

// InsertNewSize2D 向包装器中插入指定ID和尺寸的矩形
// 返回是否插入/包装成功
func (p *Packer) InsertNewSize2D(id, width, height int) bool {
	result := p.Insert(NewSize2DByID(id, width, height))
	if p.Online && len(result) != 0 {
		return false
	}
	return true
}

// SetSorter 设置用于packing的排序函数和排序顺序
// 参数:
//
//	compare - 用于比较两个尺寸大小的函数
//	reverse - 是否启用反向排序
//
// 默认比较函数为 SortArea
func (p *Packer) SetSorter(compare SortFunc, reverse bool) {
	p.sortFunc = compare
	p.sortRev = reverse
}
func (p *Packer) SetPadding(padding int) {
	p.padding = padding
}

// GetPackedRects 获取所有已成功包装的矩形
// 返回:
//
//	已包装矩形的切片(由内部管理，如需修改请复制)
func (p *Packer) GetPackedRects() []Rect2D {
	return p.algo.GetPackedRects()
}

// GetUnpackedRects 获取所有暂存但未包装的矩形尺寸
// 返回:
//
//	未包装尺寸的切片(由内部管理，如需修改请复制)
func (p *Packer) GetUnpackedRects() []Size2D {
	return p.unpackedSize2Ds
}

// GetAreaUsedRate 计算当前空间利用率
// 参数:
//
//	current - true:计算当前区域使用率 false:计算最大可能区域使用率
//
// 返回:
//
//	空间利用率(0.0-1.0)
func (p *Packer) GetAreaUsedRate(current bool) float64 {
	if current {
		size := p.MinSize()
		return float64(p.algo.GetUsedArea()) / float64(size.Width*size.Height)
	}
	return p.algo.GetAreaUsedRate()
}

// Reset 重置包装器状态(保留配置)
// 清除所有已包装和暂存的矩形
func (p *Packer) Reset() {
	size := p.algo.MaxSize()
	p.algo.Reset(size.Width, size.Height)
	p.unpackedSize2Ds = p.unpackedSize2Ds[:0]
}

// ResetMaxSize 重置包最大尺寸
// 清除所有已包装和暂存的矩形，并重置最大尺寸
// 参数:
//
//	maxWidth - 新的最大宽度(必须大于0)
//	maxHeight - 新的最大高度(必须大于0)
//
// 返回:
//
//	true: 重置成功 false: 重置失败(可能是因为参数无效)
func (p *Packer) ResetMaxSize(maxWidth, maxHeight int) bool {
	if maxWidth <= 0 || maxHeight <= 0 {
		return false
	}
	p.algo.Reset(maxWidth, maxHeight)
	p.unpackedSize2Ds = p.unpackedSize2Ds[:0]
	return true
}

// Pack 尝试打包所有暂存的矩形
// 返回:
//
//	true: 全部打包成功 false: 部分失败(可通过Unpacked获取失败尺寸)
func (p *Packer) Pack() bool {
	if len(p.unpackedSize2Ds) == 0 {
		return true
	}
	if p.sortFunc != nil {
		if p.sortRev {
			slices.SortFunc(p.unpackedSize2Ds, func(a, b Size2D) int {
				return p.sortFunc(b, a)
			})
		} else {
			slices.SortFunc(p.unpackedSize2Ds, p.sortFunc)
		}
	} else if p.sortRev {
		slices.Reverse(p.unpackedSize2Ds)
	}
	failedPackedSize2Ds := p.algo.Insert(p.padding, p.unpackedSize2Ds...)

	if len(failedPackedSize2Ds) == 0 {
		p.unpackedSize2Ds = p.unpackedSize2Ds[:0]
		return true
	}
	p.unpackedSize2Ds = failedPackedSize2Ds
	return false
}

// Shrink 自动布局收缩
//
// 适用场景:
//  1. 优化空间利用率
//
// 返回:
//
//	true: 收缩成功 false: 收缩失败(可能是因为没有足够的空间)
func (p *Packer) Shrink() bool {
	if len(p.unpackedSize2Ds) != 0 {
		//有装不下的矩形代表没有空间了
		return false
	}
	totalArea := p.algo.GetUsedArea()
	if totalArea == 0 {
		return false
	}

	// 获取当前已打包的所有矩形
	rects := p.algo.GetPackedRects()
	if len(rects) == 0 {
		return false
	}

	// 保存原始尺寸以便失败时恢复
	origSize := p.algo.MaxSize()

	// 计算最大的矩形宽度和高度
	maxWidth, maxHeight := 0, 0
	for _, rect := range rects {
		maxWidth = max(maxWidth, rect.Width)
		maxHeight = max(maxHeight, rect.Height)
	}

	// 考虑填充的影响
	if p.padding > 0 {
		maxWidth += p.padding * 2
		maxHeight += p.padding * 2
	}

	// 计算理论最小正方形边长（基于总面积的平方根，增加20%空间作为初始估计）
	estimatedSize := int(float64(totalArea) * 1.2)
	initialSize := int(math.Sqrt(float64(estimatedSize)))

	// 确保初始尺寸至少能容纳最大的单个矩形
	initialSize = max(initialSize, maxWidth)
	initialSize = max(initialSize, maxHeight)

	sizes := make([]Size2D, 0, len(rects))
	for _, rect := range rects {
		sizes = append(sizes, rect.Size2D)
	}

	// 设置搜索范围
	minSize := initialSize
	maxSize := initialSize * 2 // 开始时设置一个较大的上限

	// 先测试最小尺寸是否可行
	p.algo.Reset(minSize, minSize)
	failed := p.algo.Insert(p.padding, slices.Clone(sizes)...)
	successful := len(failed) == 0

	// 如果最小尺寸就能打包成功，直接返回
	if successful {
		p.unpackedSize2Ds = p.unpackedSize2Ds[:0]
		return true
	}
	// 如果最小尺寸不行，尝试更大的尺寸
	// 先找到一个能成功的尺寸作为上限
	for !successful && maxSize < 10000 { // 设置一个合理的上限
		maxSize *= 2
		p.algo.Reset(maxSize, maxSize)
		failed = p.algo.Insert(p.padding, slices.Clone(sizes)...)
		successful = len(failed) == 0
	}
	if !successful {
		// 恢复原始尺寸
		p.algo.Reset(origSize.Width, origSize.Height)
		p.algo.Insert(p.padding, sizes...)
		return false
	}
	// 使用二分查找找到最小的正方形尺寸
	bestSize := maxSize
	for minSize < maxSize-1 {
		midSize := (minSize + maxSize) / 2
		p.algo.Reset(midSize, midSize)
		failed = p.algo.Insert(p.padding, slices.Clone(sizes)...)
		successful = len(failed) == 0
		if successful {
			maxSize = midSize
			bestSize = midSize
		} else {
			minSize = midSize
		}
	}

	// 尝试优化宽高比
	optimalWidth := bestSize
	optimalHeight := bestSize

	// 尝试减小高度
	for h := bestSize - 1; h >= bestSize/2; h-- {
		p.algo.Reset(optimalWidth, h)
		failed = p.algo.Insert(p.padding, slices.Clone(sizes)...)
		successful = len(failed) == 0

		if successful {
			optimalHeight = h
		} else {
			break
		}
	}

	// 尝试减小宽度
	for w := bestSize - 1; w >= bestSize/2; w-- {
		p.algo.Reset(w, optimalHeight)
		failed = p.algo.Insert(p.padding, slices.Clone(sizes)...)
		successful = len(failed) == 0
		if successful {
			optimalWidth = w
		} else {
			break
		}
	}

	// 最后一次尝试使用最佳尺寸
	p.algo.Reset(optimalWidth, optimalHeight)
	failed = p.algo.Insert(p.padding, slices.Clone(sizes)...)
	successful = len(failed) == 0

	if successful {
		p.unpackedSize2Ds = p.unpackedSize2Ds[:0]
		return true
	} else {
		// 恢复原始尺寸
		fmt.Println("优化空间失败")
		p.algo.Reset(origSize.Width, origSize.Height)
		p.algo.Insert(p.padding, slices.Clone(sizes)...)
		return false
	}
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
	case Guillotine:
		p.algo = newGuillotine(maxWidth, maxHeight, heuristic)
	default:
		var a algorithmBase
		a.maxHeight = maxHeight
		a.maxWidth = maxWidth
		p.sortFunc = nil
		p.algo = &a
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
	packer, _ := NewPacker(DefaultSize, DefaultSize, MaxRectsBAF)
	return packer
}
