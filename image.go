package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"rectpack2d/rectpack"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/maruel/natural"
)

// GetImageBBox 检测并裁剪图像的透明区域，返回非透明区域的边界
func GetImageBBox(img image.Image, alphaThreshold uint32) image.Rectangle {
	bounds := img.Bounds()
	if bounds.Empty() {
		return image.Rectangle{}
	}
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	found := false
	switch src := img.(type) {
	case *image.RGBA:
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
		return bounds // 图像完全透明
	}
	// 创建非透明区域的边界矩形
	return image.Rect(minX, minY, maxX+1, maxY+1)
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
	// 创建互斥锁保护对errChan的并发访问
	var mu sync.Mutex
	// 使用Parallel函数并行处理图片
	Parallel(0, len(paths), func(i int) {
		path := paths[i]
		file, err := os.Open(path)
		if err != nil {
			mu.Lock()
			errChan <- err
			mu.Unlock()
			return
		}
		if options.IsTrimTransparent {
			// 完全解码图片以分析透明区域
			src, err := imaging.Decode(file)
			file.Close()
			if err != nil {
				mu.Lock()
				errChan <- fmt.Errorf("无法解码图片 %s: %v", path, err)
				mu.Unlock()
				return
			}
			// 获取原始尺寸
			origBounds := src.Bounds()
			sourceRects[i] = origBounds
			// 获取透明边界区域
			trimRect := GetImageBBox(src, options.TransparencyThreshold)
			sizes[i] = rectpack.NewSize2DByID(i, trimRect.Dx(), trimRect.Dy())
			sourceRects[i] = trimRect
		} else {
			// 只解码图片头部以获取尺寸信息
			cfg, _, err := image.DecodeConfig(file)
			file.Close()
			if err != nil {
				mu.Lock()
				errChan <- fmt.Errorf("无法解码图片 %s: %v", path, err)
				mu.Unlock()
				return
			}
			// 创建尺寸对象，使用索引作为ID
			sizes[i] = rectpack.NewSize2DByID(i, cfg.Width, cfg.Height)
		}
	})

	// 检查是否有错误
	close(errChan)
	for err := range errChan {
		if err != nil {
			return nil, nil, err
		}
	}

	return sizes, sourceRects, nil
}

// readImageFiles 读取目录中的所有图片文件并返回它们的尺寸
func readImageFiles() ([]rectpack.Size2D, []string, []image.Rectangle) {
	// 确保输入目录存在
	if _, err := os.Stat(options.InputDir); os.IsNotExist(err) {
		panic(fmt.Errorf("输入目录 %s 不存在", options.InputDir))
	}
	pattern := filepath.Join(options.InputDir, "*.png")
	imagePaths, _ := filepath.Glob(pattern)

	// 是否按文件名排序
	if options.IsFilesSort {
		sort.Sort(natural.StringSlice(imagePaths))
	}
	if len(imagePaths) == 0 {
		panic(fmt.Errorf("输入目录 %s 中没有找到任何图片文件", options.InputDir))
	}
	fmt.Printf("找到 %d 个图片文件\n", len(imagePaths))
	if options.IsTrimTransparent {
		fmt.Println("已开启透明区域裁切...")
	}
	size2Ds, sourceRects, err := processImages(imagePaths)
	if err != nil {
		panic(err)
	}
	fmt.Printf("预先处理 %d 个图片文件\n", len(size2Ds))
	return size2Ds, imagePaths, sourceRects
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
}

// CreateAtlasImage 创建图集图像
func CreateAtlasImage(packer *rectpack.Packer, imagePaths []string, sourceRects []image.Rectangle) (*image.NRGBA, map[string]SpriteInfo, error) {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.CreateAtlasImageTime += elapsed
		}()
	}
	// 获取图集所需的最终尺寸
	atlasSize := packer.MinSize()
	if options.PowerOfTwo {
		atlasSize.Width = nextPowerOfTwo(atlasSize.Width)
		atlasSize.Height = nextPowerOfTwo(atlasSize.Height)
	}

	spriteInfoMapping := make(map[string]SpriteInfo, len(packer.GetPackedRects()))
	dstImage := imaging.New(atlasSize.Width, atlasSize.Height, color.NRGBA{0, 0, 0, 0})
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
			srcImage, err := imaging.Decode(file)
			file.Close()
			if err != nil {
				errChan <- fmt.Errorf("%s: %v", path, err)
				return
			}

			origBounds := srcImage.Bounds()
			srcRect := sourceRects[r.ID]

			// 检查是否需要旋转
			isRotated := false
			if packer.GetIdMapToRotateCount()[r.ID]&1 != 0 { // 奇数说明旋转了
				isRotated = true
				srcImage = imaging.Rotate270(srcImage)
				origHeight := origBounds.Dy()
				newMinX := origHeight - srcRect.Min.Y - srcRect.Dy()
				newMinY := srcRect.Min.X
				newWidth := srcRect.Dy()
				newHeight := srcRect.Dx()
				srcRect = image.Rect(newMinX, newMinY, newMinX+newWidth, newMinY+newHeight)
			}

			origBounds = srcImage.Bounds() // 旋转后的
			// 创建精灵信息
			spriteInfo := SpriteInfo{}
			spriteInfo.Filename = filepath.Base(path)
			spriteInfo.Region.X = r.X
			spriteInfo.Region.Y = r.Y
			spriteInfo.Region.W = r.Width
			spriteInfo.Region.H = r.Height
			spriteInfo.Rotated = isRotated
			spriteInfo.SourceSize.W = origBounds.Dx()
			spriteInfo.SourceSize.H = origBounds.Dy()

			// 检查是否进行了裁剪
			isTrimmed := srcRect.Min.X > 0 || srcRect.Min.Y > 0 ||
				srcRect.Dx() < origBounds.Dx() || srcRect.Dy() < origBounds.Dy()
			if isTrimmed {
				spriteInfo.Trimmed = true
				spriteInfo.SourceRect.X = srcRect.Min.X
				spriteInfo.SourceRect.Y = srcRect.Min.Y
				spriteInfo.SourceRect.W = srcRect.Dx()
				spriteInfo.SourceRect.H = srcRect.Dy()
			}

		
			dstRect := image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
			
			mu.Lock()
			// 绘制图片
			//dstImage 目标图像
			//dstRect 源图像在目标图像中的位置 左上坐标,右下坐标
			//srcImage 源图像
			//sourceRect.Min 源矩形的左上角坐标
			//draw.Src 绘制操作的选项，这里是使用源图像的原始像素
			draw.Draw(dstImage, dstRect, srcImage, srcRect.Min, draw.Src)
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
