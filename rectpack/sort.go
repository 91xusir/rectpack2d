package rectpack

import "cmp"

// SortFunc 定义矩形尺寸比较函数的原型
// 返回值:
//   -1: a < b
//    0: a == b
//    1: a > b
type SortFunc func(a, b Size) int

// SortArea 按矩形面积降序排序(从大到小)
func SortArea(a, b Size) int {
	return cmp.Compare(b.Area(), a.Area())
}

// SortPerimeter 按矩形周长降序排序(从大到小)
func SortPerimeter(a, b Size) int {
	return cmp.Compare(b.Perimeter(), a.Perimeter())
}

// SortDiff 按矩形宽高差降序排序(从大到小)
func SortDiff(a, b Size) int {
	return cmp.Compare(abs(b.Width-b.Height), abs(a.Width-a.Height))
}

// SortMinSide 按矩形最短边降序排序(从大到小)
func SortMinSide(a, b Size) int {
	return cmp.Compare(b.MinSide(), a.MinSide())
}

// SortMaxSide 按矩形最长边降序排序(从大到小)
func SortMaxSide(a, b Size) int {
	return cmp.Compare(b.MaxSide(), a.MaxSide())
}

// SortRatio 按矩形宽高比降序排序(从大到小)
func SortRatio(a, b Size) int {
	return cmp.Compare(b.Ratio(), a.Ratio())
}
