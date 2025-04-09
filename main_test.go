package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"image"
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
	}
	options2 := Options{
		UnpackPath:            "output\\atlases.json",
		InputDir:              "input2",
		OutputDir:             "output",
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
				runBenchmark(&options2, algo+"_"+variant)
			})
		}
	}

}
func runBenchmark(options *Options, name string) {
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

	// 读取输入目录中的图片文件
	size2Ds, imagePaths, sourceRects := readImageFiles(options)

	pakerList := make([]*rectpack.Packer, 0)
	// 创建打包器并打包当前批次的图片
	packer := packing(size2Ds, options)
	// 输出当前打包结果
	outputResult(packer)

	pakerList = append(pakerList, packer)

	for unpackedRects := packer.GetUnpackedRects(); len(unpackedRects) > 0; {
		p := packing(unpackedRects, options)
		outputResult(p)
		pakerList = append(pakerList, p)
		unpackedRects = p.GetUnpackedRects()
	}

	atlasList := make([]*image.NRGBA, 0)
	multiSpiteInfo := make([]map[string]SpriteInfo, 0)

	for atlasIndex, packer := range pakerList {
		atlasImage, spriteInfoMapping, err := CreateAtlasImage(packer, imagePaths, sourceRects)
		if err != nil {
			fmt.Printf("生成图集 #%d 失败: %v\n", atlasIndex, err)
			continue
		}
		atlasList = append(atlasList, atlasImage)
		multiSpiteInfo = append(multiSpiteInfo, spriteInfoMapping)
	}
	atlasImagePaths := make([]string, 0)

	// 确保输出目录存
	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}
	for i, a := range atlasList {
		var imageName string
		if len(atlasList) == 1 {
			imageName = name + "atlas.png"
		} else {
			imageName = fmt.Sprintf(name+"atlas_%d.png", i)
		}
		outputPath := filepath.Join(options.OutputDir, imageName)
		atlasImagePaths = append(atlasImagePaths, outputPath)
		// 保存图集图像
		file, _ := os.Create(outputPath)
		imaging.Encode(file, a, imaging.PNG)
		file.Close()
	}
	multiAtlasJsonPath := filepath.Join(options.OutputDir, name+".json")
	if err := generateMultiAtlasJSON(multiSpiteInfo, atlasImagePaths, multiAtlasJsonPath); err != nil {
		fmt.Printf("生成JSON元数据失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("- 图集元数据: %s\n\n", multiAtlasJsonPath)
}
