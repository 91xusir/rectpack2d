package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"rectpack2d/rectpack"
	"time"
)

const (
	VERSION = "0.1.0"
)

var (
	options   Options
	debugInfo = DebugInfo{IsDebug: true}
)

type DebugInfo struct {
	IsDebug              bool
	TotalTime            time.Duration
	PackTime             time.Duration
	FileSortTime         time.Duration
	ProcessImageTime     time.Duration
	CreateAtlasImageTime time.Duration
	CreateJsonTime       time.Duration
}
type Options struct {
	InputDir              string             // 输入目录
	OutputDir             string             // 输出目录
	AtlasMaxWidth         int                // 最大宽度
	AtlasMaxHeight        int                // 最大高度
	IsFilesSort           bool               // 是否按文件名排序
	SpritePadding         int                // 填充
	IsAllowRotate         bool               // 是否允许旋转
	IsTrimTransparent     bool               // 是否修剪透明部分
	TransparencyThreshold uint32             //透明度阈值
	IsSameDetection       bool               //相同检测
	IsAutoSize            bool               //是否自动收缩
	Algorithm             rectpack.Heuristic // 算法
}

// SpriteInfo 存储精灵图的信息
type SpriteInfo struct {
	Filename string `json:"filename"`
	Region   struct {
		X int `json:"x"`
		Y int `json:"y"`
		W int `json:"w"`
		H int `json:"h"`
	} `json:"region"`
	SourceSize struct {
		W int `json:"w"`
		H int `json:"h"`
	} `json:"sourceSize"`
	SourceRect struct {
		X int `json:"x"`
		Y int `json:"y"`
		W int `json:"w"`
		H int `json:"h"`
	} `json:"sourceRect,omitempty"`
	Trimmed bool `json:"trimmed"`
	Rotated bool `json:"rotated"`
}

// MultiAtlasData 存储多个图集的信息
type MultiAtlasData struct {
	Meta struct {
		Version   string `json:"version"`
		Timestamp string `json:"timestamp"`
	} `json:"meta"`
	Atlases []struct {
		Atlas  string                `json:"atlas"`
		Sprites map[string]SpriteInfo `json:"sprites"`
		Size   struct {
			W int `json:"w"`
			H int `json:"h"`
		} `json:"size"`
	} `json:"atlases"`
}

// generateMultiAtlasJSON 生成包含多个图集信息的JSON元数据
func generateMultiAtlasJSON(atlasMappings []map[string]SpriteInfo, atlasImagePaths []string, outputPath string) error {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.CreateJsonTime = elapsed
		}()
	}
	// 创建多图集数据结构
	multiAtlasData := MultiAtlasData{
		Meta: struct {
			Version   string `json:"version"`
			Timestamp string `json:"timestamp"`
		}{
			Version:   VERSION,
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		},
		Atlases: make([]struct {
			Atlas  string                `json:"atlas"`
			Sprites map[string]SpriteInfo `json:"sprites"`
			Size   struct {
				W int `json:"w"`
				H int `json:"h"`
			} `json:"size"`
		}, len(atlasMappings)),
	}

	// 填充每个图集的信息
	for i, mapping := range atlasMappings {
		atlas := &multiAtlasData.Atlases[i]
		atlas.Atlas = filepath.Base(atlasImagePaths[i])
		atlas.Sprites = make(map[string]SpriteInfo)

		// 计算图集的总尺寸
		var maxWidth, maxHeight int
		for _, spriteInfo := range mapping {
			right := spriteInfo.Region.X + spriteInfo.Region.W
			bottom := spriteInfo.Region.Y + spriteInfo.Region.H
			if right > maxWidth {
				maxWidth = right
			}
			if bottom > maxHeight {
				maxHeight = bottom
			}
			// 添加到帧集合
			atlas.Sprites[spriteInfo.Filename] = spriteInfo
		}

		// 设置图集尺寸
		atlas.Size.W = maxWidth
		atlas.Size.H = maxHeight
	}

	// 将数据编码为JSON
	jsonData, err := json.MarshalIndent(multiAtlasData, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(outputPath, jsonData, 0644)
}

// outputResult 输出打包结果
func outputResult(packer *rectpack.Packer) {
	rects := packer.GetPackedRects()
	size := packer.MinSize()
	fmt.Printf("打包区域大小: %dx%d\n", size.Width, size.Height)
	fmt.Printf("空间利用率: %.2f%%\n", packer.GetAreaUsedRate(true)*100)
	fmt.Printf("已打包矩形数量: %d\n", len(rects))
	fmt.Printf("未打包矩形数量: %d\n\n", len(packer.GetUnpackedRects()))
}

func packing(sizes []rectpack.Size2D) *rectpack.Packer {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.PackTime += elapsed
		}()
	}
	packer, err := rectpack.NewPacker(options.AtlasMaxWidth, options.AtlasMaxHeight, options.Algorithm)
	if err != nil {
		fmt.Printf("创建打包器失败: %v\n", err)
		os.Exit(1)
	}
	packer.AllowRotate(options.IsAllowRotate)
	packer.SetPadding(options.SpritePadding)
	packer.Insert(sizes...)
	successful := packer.Pack()
	if successful && options.IsAutoSize {
		fmt.Println("空间自动收缩优化...")
		packer.Shrink()
	}
	if !successful {
		fmt.Println("警告: 部分图片无法打包到指定尺寸的图集中")
	}
	return packer
}

func main() {
	if debugInfo.IsDebug {
		start := time.Now() // 记录开始时间
		defer func() {
			elapsed := time.Since(start) // 计算耗时
			debugInfo.TotalTime = elapsed
			fmt.Printf("图片预处理(裁切等)耗时: %v\n", debugInfo.ProcessImageTime)
			fmt.Printf("文件排序耗时: %v\n", debugInfo.FileSortTime)
			fmt.Printf("算法耗时:%v\n", debugInfo.PackTime)
			fmt.Printf("图集创建耗时:%v\n", debugInfo.CreateAtlasImageTime)
			fmt.Printf("JSON元数据创建耗时:%v\n", debugInfo.CreateJsonTime)
			fmt.Printf("总耗时:%v\n", debugInfo.TotalTime)
		}()
	}
	// 定义命令行参数
	inputDirPtr := flag.String("input", "input", "输入目录")
	outputDirPtr := flag.String("output", "output", "输出目录")
	paddingPtr := flag.Int("padding", 0, "填充")
	trimPtr := flag.Bool("trim", false, "修剪透明部分")
	thresholdPtr := flag.Uint("threshold", 0, "透明度阈值")
	sortPtr := flag.Bool("sort", true, "按文件名排序")
	widthPtr := flag.Int("width", 1024, "打包区域宽度")
	heightPtr := flag.Int("height", 1024, "打包区域高度")
	rotationPtr := flag.Bool("rotate", false, "允许矩形旋转")
	algorithmPtr := flag.String("algorithm", "MaxRects", "打包算法 (MaxRects, Guillotine, Skyline)")
	variantPtr := flag.String("variant", "BestAreaFit", "打包算法变体 (BestShortSideFit, BestLongSideFit, BestAreaFit)")
	autoSizePtr := flag.Bool("auto-size", false, "启用自动布局区域收缩优化")
	flag.Parse()

	// 创建选项对象
	options = Options{
		InputDir:              *inputDirPtr,
		OutputDir:             *outputDirPtr,
		SpritePadding:         *paddingPtr,
		IsTrimTransparent:     *trimPtr,
		TransparencyThreshold: (uint32)(*thresholdPtr),
		AtlasMaxWidth:         *widthPtr,
		AtlasMaxHeight:        *heightPtr,
		IsAllowRotate:         *rotationPtr,
		IsFilesSort:           *sortPtr,
		IsSameDetection:       false,
		IsAutoSize:            *autoSizePtr,
		Algorithm:             rectpack.ResolveAlgorithm(*algorithmPtr, *variantPtr),
	}

	// 创建输出目录（如果不存在）
	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 读取输入目录中的图片文件
	size2Ds, imagePaths, sourceRects, err := readImageFiles()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	pakers := make([]*rectpack.Packer, 0)
	// 创建打包器并打包当前批次的图片
	packer := packing(size2Ds)
	// 输出当前图集的打包结果
	outputResult(packer)
	pakers = append(pakers, packer)
	for unpackedRects := packer.GetUnpackedRects(); len(unpackedRects) > 0; {
		p := packing(unpackedRects)
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
			os.Exit(1)
		}
		// 保存图集图像
		file, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("创建文件失败: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		if err := png.Encode(file, atlasImages[0]); err != nil {
			fmt.Printf("保存图像失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		for i, atlasImage := range atlasImages {
			outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("atlas_%d.png", i))
			atlasImagePaths = append(atlasImagePaths, outputPath)
			// 确保输出目录存在，而不是将输出路径创建为目录
			if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
				fmt.Printf("创建输出目录失败: %v\n", err)
				os.Exit(1)
			}
			// 保存图集图像
			file, err := os.Create(outputPath)
			if err != nil {
				fmt.Printf("创建文件失败: %v\n", err)
				os.Exit(1)
			}
			defer file.Close()
			if err := png.Encode(file, atlasImage); err != nil {
				fmt.Printf("保存图像失败: %v\n", err)
				os.Exit(1)
			}
		}
	}
	// 生成当前图集的JSON元数据

	multiAtlasJsonPath := filepath.Join(options.OutputDir, "atlases.json")
	if err := generateMultiAtlasJSON(multiSpiteInfo, atlasImagePaths, multiAtlasJsonPath); err != nil {
		fmt.Printf("生成JSON元数据失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n成功生成 %d 个图集:\n", len(atlasImages))
	for i, path := range atlasImagePaths {
		fmt.Printf("- 图集 #%d: %s\n", i+1, path)
	}
	fmt.Printf("- 多图集元数据: %s\n\n", multiAtlasJsonPath)
}
