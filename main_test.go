package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"rectpack2d/rectpack"
	"testing"
	"time"
)

func TestPackAll(t *testing.T) {
	algos := map[string][]string{
		"MaxRects":   {"BestShortSideFit", "BottomLeft", "ContactPoint", "BestLongSideFit", "BestAreaFit"},
		"Guillotine": {"BestAreaFit", "BestShortSideFit", "BestLongSideFit", "WorstAreaFit", "WorstShortSideFit", "WorstLongSideFit"},
		"Skyline":    {"BottomLeft", "MinWaste"},
	}
	options2 := Options{
		UnpackPath:            "output\atlases.json",
		InputDir:              "input",
		OutputDir:             "ouput",
		SpritePadding:         0,
		IsTrimTransparent:     true,
		TransparencyThreshold: (uint32)(0),
		AtlasMaxWidth:         4096,
		AtlasMaxHeight:        4096,
		IsAllowRotate:         true,
		IsFilesSort:           true,
		IsSameDetection:       false,
		IsAutoSize:            true,
	}
	// 遍历所有算法
	for algo, variants := range algos {
		for _, variant := range variants {
			t.Run(algo+"_"+variant, func(t *testing.T) {
				heuristic := rectpack.ResolveAlgorithm(algo, variant)
				options2.Algorithm = heuristic
				runBenchmark(&options2)
			})
		}
	}

}
func runBenchmark(options* Options) {
	start := time.Now() // 记录开始时间
	defer func() {
		elapsed := time.Since(start) // 计算耗时
		debugInfo.TotalTime = elapsed
		fmt.Printf("图片预处理(裁切等)耗时: %v\n", debugInfo.ProcessImageTime)
		fmt.Printf("文件排序耗时: %v\n", debugInfo.FileSortTime)
		fmt.Printf("算法耗时:%v\n", debugInfo.PackTime)
		fmt.Printf("图集创建耗时:%v\n", debugInfo.CreateAtlasImageTime)
		fmt.Printf("JSON元数据创建耗时:%v\n", debugInfo.CreateJsonTime)
		fmt.Printf("总耗时:%v\n\n", debugInfo.TotalTime)
	}()
	// 创建输出目录（如果不存在）
	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 读取输入目录中的图片文件
	size2Ds, imagePaths, sourceRects := readImageFiles()

	pakers := make([]*rectpack.Packer, 0)
	// 创建打包器并打包当前批次的图片
	packer := packing(size2Ds,options)
	// 输出当前图集的打包结果
	outputResult(packer)
	pakers = append(pakers, packer)
	for unpackedRects := packer.GetUnpackedRects(); len(unpackedRects) > 0; {
		p := packing(unpackedRects,options)
		outputResult(p)
		pakers = append(pakers, p)
		unpackedRects = p.GetUnpackedRects()
	}
	atlasImages := make([]image.Image, 0)
	multiSpiteInfo := make([]map[string]SpriteInfo, 0)

	for atlasIndex, packer := range pakers {
		atlasImage, spriteInfoMapping, err := CreateAtlasImage(packer, imagePaths, sourceRects)
		if err != nil {
			fmt.Printf("生成图集 #%d 失败: %v\n", atlasIndex, err)
			continue
		}
		atlasImages = append(atlasImages, atlasImage)
		multiSpiteInfo = append(multiSpiteInfo, spriteInfoMapping)
	}
	atlasImagePaths := make([]string, 0)
	if len(atlasImages) == 1 {
		outputPath := filepath.Join(options.OutputDir, "atlas.png")
		atlasImagePaths = append(atlasImagePaths, outputPath)
		// 确保输出目录存在，而不是将输出路径创建为目录
		if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
			fmt.Printf("创建输出目录失败: %v\n", err)
		}
		// 保存图集图像
		file, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("创建文件失败: %v\n", err)
		}
		defer file.Close()
		if err := png.Encode(file, atlasImages[0]); err != nil {
			fmt.Printf("保存图像失败: %v\n", err)
		}
	} else {
		for i, atlasImage := range atlasImages {
			outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("atlas_%d.png", i))
			atlasImagePaths = append(atlasImagePaths, outputPath)
			// 确保输出目录存在，而不是将输出路径创建为目录
			if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
				fmt.Printf("创建输出目录失败: %v\n", err)
			}
			// 保存图集图像
			file, err := os.Create(outputPath)
			if err != nil {
				fmt.Printf("创建文件失败: %v\n", err)
			}
			defer file.Close()
			if err := png.Encode(file, atlasImage); err != nil {
				fmt.Printf("保存图像失败: %v\n", err)
			}
		}
	}
	// 生成当前图集的JSON元数据

	multiAtlasJsonPath := filepath.Join(options.OutputDir, "atlases.json")
	if err := generateMultiAtlasJSON(multiSpiteInfo, atlasImagePaths, multiAtlasJsonPath); err != nil {
		fmt.Printf("生成JSON元数据失败: %v\n", err)
	}

	fmt.Printf("\n成功生成 %d 个图集:\n", len(atlasImages))
	for i, path := range atlasImagePaths {
		fmt.Printf("- 图集 #%d: %s\n", i+1, path)
	}
	fmt.Printf("- 多图集元数据: %s\n\n", multiAtlasJsonPath)

}
