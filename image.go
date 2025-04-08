package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"rectpack2d/rectpack"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

// trimTransparentArea 检测并裁剪图像的透明区域，返回非透明区域的边界
func trimTransparentArea(img image.Image) (image.Rectangle, bool) {
	alphaThreshold := options.TransparencyThreshold
	bounds := img.Bounds()
	if bounds.Empty() {
		return bounds, false
	}
	// 初始化边界值
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	found := false
	// 针对不同图像类型优化访问方式
	switch src := img.(type) {
	case *image.RGBA:
		// 优化RGBA图像处理
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			i := src.PixOffset(bounds.Min.X, y)
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if src.Pix[i+3] > uint8(alphaThreshold) { // 直接访问alpha通道
					found = true
					if x < minX {
						minX = x
					}
					if y < minY {
						minY = y
					}
					if x > maxX {
						maxX = x
					}
					if y > maxY {
						maxY = y
					}
				}
				i += 4
			}
		}
	case *image.NRGBA:
		// 优化NRGBA图像处理
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			i := src.PixOffset(bounds.Min.X, y)
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if src.Pix[i+3] > uint8(alphaThreshold) { // 直接访问alpha通道
					found = true
					if x < minX {
						minX = x
					}
					if y < minY {
						minY = y
					}
					if x > maxX {
						maxX = x
					}
					if y > maxY {
						maxY = y
					}
				}
				i += 4
			}
		}
	default:
		// 通用处理方式
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				a8 := a >> 8             // RGBA()返回的是16bit，转换为8bit
				if a8 > alphaThreshold { // 非完全透明的像素
					found = true
					if x < minX {
						minX = x
					}
					if y < minY {
						minY = y
					}
					if x > maxX {
						maxX = x
					}
					if y > maxY {
						maxY = y
					}
				}
			}
		}
	}
	if !found {
		return bounds, false // 图像完全透明
	}
	// 创建非透明区域的边界矩形
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

func sortFileBynName(paths []string) {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.FileSortTime += elapsed
		}()
	}
	// 按文件名排序（支持数字排序）
	slices.SortFunc(paths, func(i, j string) int {
		nameI := filepath.Base(i)
		nameJ := filepath.Base(j)
		// 尝试提取文件名中的数字部分
		numI, errI := strconv.Atoi(strings.TrimSuffix(nameI, filepath.Ext(nameI)))
		numJ, errJ := strconv.Atoi(strings.TrimSuffix(nameJ, filepath.Ext(nameJ)))
		// 如果两个文件名都是纯数字，按数字大小排序
		if errI == nil && errJ == nil {
			if numI < numJ {
				return -1
			} else if numI > numJ {
				return 1
			}
			return 0
		}
		// 否则按字符串排序
		if nameI < nameJ {
			return -1
		} else if nameI > nameJ {
			return 1
		}
		return 0
	})
}

func processImages(paths []string) ([]rectpack.Size2D, []image.Rectangle, error) {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.ProcessImageTime += elapsed
		}()
	}
	
	sourceRects := make([]image.Rectangle, len(paths))
	sizes := make([]rectpack.Size2D, len(paths))
	
	// 创建错误通道
	errChan := make(chan error, len(paths))
	// 创建等待组
	var wg sync.WaitGroup
	
	// 添加并发控制，避免创建过多goroutine
	maxWorkers := runtime.NumCPU()
	semaphore := make(chan struct{}, maxWorkers)
	
	for i, path := range paths {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量
		
		go func(idx int, imgPath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量
			
			file, err := os.Open(imgPath)
			if err != nil {
				errChan <- err
				return
			}
			
			if options.IsTrimTransparent {
				// 完全解码图片以分析透明区域
				src, _, err := image.Decode(file)
				file.Close()
				if err != nil {
					errChan <- fmt.Errorf("无法解码图片 %s: %v", imgPath, err)
					return
				}
				// 获取原始尺寸
				origBounds := src.Bounds()
				sourceRects[idx] = origBounds
				// 裁剪透明区域
				trimRect, hasTrimmed := trimTransparentArea(src)
				if hasTrimmed && (trimRect.Dx() < origBounds.Dx() || trimRect.Dy() < origBounds.Dy()) {
					// 使用裁剪后的尺寸
					sizes[idx] = rectpack.NewSize2DByID(idx, trimRect.Dx(), trimRect.Dy())
					sourceRects[idx] = trimRect
				} else {
					// 使用原始尺寸
					sizes[idx] = rectpack.NewSize2DByID(idx, origBounds.Dx(), origBounds.Dy())
				}
			} else {
				// 只解码图片头部以获取尺寸信息
				cfg, _, err := image.DecodeConfig(file)
				file.Close()
				if err != nil {
					errChan <- fmt.Errorf("无法解码图片 %s: %v", imgPath, err)
					return
				}
				// 创建尺寸对象，使用索引作为ID
				sizes[idx] = rectpack.NewSize2DByID(idx, cfg.Width, cfg.Height)
			}
		}(i, path)
	}
	
	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)
	
	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return nil, nil, err
		}
	}
	
	return sizes, sourceRects, nil
}

// readImageFiles 读取目录中的所有图片文件并返回它们的尺寸
func readImageFiles() ([]rectpack.Size2D, []string, []image.Rectangle, error) {
	// 确保输入目录存在
	if _, err := os.Stat(options.InputDir); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("输入目录 %s 不存在", options.InputDir)
	}
	// 获取所有PNG图片文件
	pattern := filepath.Join(options.InputDir, "*.png")
	imagePaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, nil, err
	}
	// 按文件名排序
	if options.IsFilesSort {
		sortFileBynName(imagePaths)
	}
	if len(imagePaths) == 0 {
		return nil, nil, nil, fmt.Errorf("在 %s 目录中没有找到PNG图片", options.InputDir)
	}
	fmt.Printf("找到 %d 个图片文件\n", len(imagePaths))
	if options.IsTrimTransparent {
		fmt.Println("已开启透明区域裁切...")
	}
	size2Ds, sourceRects, err := processImages(imagePaths)
	fmt.Printf("预先处理 %d 个图片文件\n", len(size2Ds))
	return size2Ds, imagePaths, sourceRects, err
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
}

func rotate90(m image.Image) image.Image {
	rotate90 := image.NewNRGBA(image.Rect(0, 0, m.Bounds().Dy(), m.Bounds().Dx()))
	for x := m.Bounds().Min.Y; x < m.Bounds().Max.Y; x++ {
		for y := m.Bounds().Max.X - 1; y >= m.Bounds().Min.X; y-- {
			rotate90.Set(m.Bounds().Max.Y-x, y, m.At(y, x))
		}
	}
	return rotate90
}
func rotate270(m image.Image) image.Image {
	rotate270 := image.NewNRGBA(image.Rect(0, 0, m.Bounds().Dy(), m.Bounds().Dx()))
	// 矩阵旋转
	for x := m.Bounds().Min.Y; x < m.Bounds().Max.Y; x++ {
		for y := m.Bounds().Max.X - 1; y >= m.Bounds().Min.X; y-- {
			// 设置像素点
			rotate270.Set(x, m.Bounds().Max.X-y, m.At(y, x))
		}
	}
	return rotate270

}

// CreateAtlasImage 创建图集图像
func CreateAtlasImage(packer *rectpack.Packer, imagePaths []string, sourceRects []image.Rectangle) (image.Image, map[string]SpriteInfo, error) {
	
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.CreateAtlasImageTime += elapsed
		}()
	}
	
	// 获取图集所需的最终尺寸
	atlasSize := packer.MinSize()
	spriteInfoMapping := make(map[string]SpriteInfo, len(packer.GetPackedRects()))
	dstImage := image.NewNRGBA(image.Rect(0, 0, atlasSize.Width, atlasSize.Height))

	// 创建互斥锁保护对dstImage和spriteInfoMapping的并发访问
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(packer.GetPackedRects()))
	
	// 添加并发控制
	maxWorkers := runtime.NumCPU()
	semaphore := make(chan struct{}, maxWorkers)

	// 遍历每个打包的矩形
	for _, rect := range packer.GetPackedRects() {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量
		
		go func(r rectpack.Rect2D) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量
			
			// 矩形的ID是路径的索引
			path := imagePaths[r.ID]
			file, err := os.Open(path)
			if err != nil {
				errChan <- fmt.Errorf("无法打开图片 %s: %v", path, err)
				return
			}
			
			srcImage, _, err := image.Decode(file)
			if err != nil {
				file.Close()
				errChan <- fmt.Errorf("无法解码图片 %s: %v", path, err)
				return
			}
			file.Close()
			
			origBounds := srcImage.Bounds()
			sourceRect := sourceRects[r.ID]
			
			// 检查是否需要旋转
			if packer.SourceRectMapW[r.ID] != r.Width { // 旋转了
				r.Rotated = true
				srcImage = imaging.Rotate270(srcImage)
				origHeight := origBounds.Dy()
				newMinX := origHeight - sourceRect.Min.Y - sourceRect.Dy()
				newMinY := sourceRect.Min.X
				newWidth := sourceRect.Dy()
				newHeight := sourceRect.Dx()
				sourceRect = image.Rect(newMinX, newMinY, newMinX+newWidth, newMinY+newHeight)
			}
			
			origBounds = srcImage.Bounds() // 旋转后的
			
			// 创建精灵信息
			spriteInfo := SpriteInfo{}
			spriteInfo.Filename = filepath.Base(path)
			spriteInfo.Region.X = r.X
			spriteInfo.Region.Y = r.Y
			spriteInfo.Region.W = r.Width
			spriteInfo.Region.H = r.Height
			spriteInfo.Rotated = r.Rotated
			spriteInfo.SourceSize.W = origBounds.Dx()
			spriteInfo.SourceSize.H = origBounds.Dy()
			
			// 检查是否进行了裁剪
			isTrimmed := sourceRect.Min.X > 0 || sourceRect.Min.Y > 0 ||
				sourceRect.Dx() < origBounds.Dx() || sourceRect.Dy() < origBounds.Dy()
			if isTrimmed {
				spriteInfo.Trimmed = true
				spriteInfo.SourceRect.X = sourceRect.Min.X
				spriteInfo.SourceRect.Y = sourceRect.Min.Y
				spriteInfo.SourceRect.W = sourceRect.Dx()
				spriteInfo.SourceRect.H = sourceRect.Dy()
			}

			// 将图片的裁剪部分绘制到目标矩形的位置
			dstRect := image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
			
			// 使用互斥锁保护对共享资源的访问
			mu.Lock()
			// 绘制图片
			//dstImage 目标图像
			//dstRect 源图像在目标图像中的位置 左上坐标,右下坐标
			//srcImage 源图像
			//sourceRect.Min 源矩形的左上角坐标
			//draw.Src 绘制操作的选项，这里是使用源图像的原始像素
			draw.Draw(dstImage, dstRect, srcImage, sourceRect.Min, draw.Src)
			spriteInfoMapping[path] = spriteInfo
			mu.Unlock()
			
		}(rect)
	}
	
	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)
	
	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return nil, nil, err
		}
	}
	
	return dstImage, spriteInfoMapping, nil
}