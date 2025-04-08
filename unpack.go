package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

func Parallel(start, end int, fn func(i int)) {
	numGoroutines := runtime.NumCPU()
	if end-start < numGoroutines {
		// 如果任务数量少于CPU核心数，直接顺序执行
		for i := start; i < end; i++ {
			fn(i)
		}
		return
	}
	var wg sync.WaitGroup
	batchSize := (end - start) / numGoroutines
	if batchSize < 1 {
		batchSize = 1
	}
	for i := start; i < end; i += batchSize {
		wg.Add(1)
		go func(from, to int) {
			defer wg.Done()
			for j := from; j < to && j < end; j++ {
				fn(j)
			}
		}(i, i+batchSize)
	}
	wg.Wait()
}

// 解包图集函数
func unpack() error {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			fmt.Printf("解包耗时: %s\n", elapsed)
		}()
	}
	if options.UnpackPath == "" {
		return fmt.Errorf("未指定解包路径")
	}

	// 读取JSON文件
	jsonData, err := os.ReadFile(options.UnpackPath)
	if err != nil {
		return fmt.Errorf("读取图集JSON文件失败: %v", err)
	}

	// 解析JSON
	var multiAtlasData MultiAtlasData
	if err := json.Unmarshal(jsonData, &multiAtlasData); err != nil {
		return fmt.Errorf("解析JSON失败: %v", err)
	}

	// 创建输出目录
	outputDir := options.OutputDir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 处理每个图集
	for _, atlas := range multiAtlasData.Atlases {
		// 获取图集图片路径
		atlasDir := filepath.Dir(options.UnpackPath)
		atlasImagePath := filepath.Join(atlasDir, atlas.Atlas)

		// 加载图集图片
		atlasFile, err := os.Open(atlasImagePath)
		if err != nil {
			return fmt.Errorf("打开图集图片失败: %v", err)
		}

		atlasImg, err := imaging.Decode(atlasFile)
		if err != nil {
			atlasFile.Close()
			return fmt.Errorf("解码图集图片失败: %v", err)
		}
		atlasFile.Close()

		// 将sprites转换为切片以便并行处理
		type spriteEntry struct {
			name   string
			sprite SpriteInfo
		}
		spriteList := make([]spriteEntry, 0, len(atlas.Sprites))
		for name, sprite := range atlas.Sprites {
			spriteList = append(spriteList, spriteEntry{name, sprite})
		}

		// 创建错误通道和互斥锁
		errChan := make(chan error, len(spriteList))
		var mu sync.Mutex

		// 并行处理每个子图
		Parallel(0, len(spriteList), func(i int) {
			entry := spriteList[i]
			name, sprite := entry.name, entry.sprite

			// 创建新图片
			subImg := imaging.New(sprite.Region.W, sprite.Region.H, color.NRGBA{0, 0, 0, 0})
			// 绘制子图
			subImg = imaging.Crop(atlasImg, image.Rect(sprite.Region.X, sprite.Region.Y, 
				sprite.Region.X+sprite.Region.W, sprite.Region.Y+sprite.Region.H))

			// 如果需要处理修剪的图片
			if sprite.Trimmed {
				// 创建一个与原始尺寸相同的透明图片
				finalImg := imaging.New(sprite.SourceSize.W, sprite.SourceSize.H, color.NRGBA{0, 0, 0, 0})
				// 将子图绘制到正确位置
				finalImg = imaging.Paste(finalImg, subImg, image.Point{sprite.SourceRect.X, sprite.SourceRect.Y})
				subImg = finalImg
			}
			// 如果需要旋转图片
			if sprite.Rotated {
				subImg = imaging.Rotate90(subImg)
			}
			// 构建保存路径
			outputPath := filepath.Join(outputDir, name)
			
			// 使用互斥锁保护文件系统操作
			mu.Lock()
			// 确保输出子目录存在
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				errChan <- fmt.Errorf("创建输出子目录失败: %v", err)
				mu.Unlock()
				return
			}
			mu.Unlock()

			outFile, err := os.Create(outputPath)
			if err != nil {
				errChan <- fmt.Errorf("创建输出文件失败: %v", err)
				return
			}
			// 保存为PNG格式
			if err := imaging.Encode(outFile, subImg, imaging.PNG); err != nil {
				outFile.Close()
				errChan <- fmt.Errorf("编码图片失败: %v", err)
				return
			}
			outFile.Close()
		})

		// 检查是否有错误
		close(errChan)
		for err := range errChan {
			if err != nil {
				return err
			}
		}
	}
	fmt.Printf("图集解包完成，输出到: %s\n", outputDir)
	return nil
}
