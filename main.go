package main

import (
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"rectpack2d/rectpack"
)

// type SpriteSize rectpack.Size

type Options struct {
	InputDir          string // 输入目录
	OutputDir         string // 输出目录
	AtlasMaxWidth     int    // 最大宽度
	AtlasMaxHeight    int    // 最大高度
	SpritePadding     int    // 填充
	IsAllowRotate     bool   // 是否允许旋转
	IsTrimTransparent bool   // 是否修剪透明部分
	Algorithm         string // 算法
}

func ResolveAlgorithm(main, variant string) rectpack.Heuristic {
	switch main {
	case "MaxRects":
		switch variant {
		case "BestShortSideFit":
			return rectpack.MaxRectsBSSF
		case "BottomLeft":
			return rectpack.MaxRectsBL
		case "ContactPoint":
			return rectpack.MaxRectsCP
		case "BestLongSideFit":
			return rectpack.MaxRectsBLSF
		case "BestAreaFit":
			return rectpack.MaxRectsBAF
		}
	case "Guillotine":
		switch variant {
		case "BestAreaFit":
			return rectpack.GuillotineBAF
		case "BestShortSideFit":
			return rectpack.GuillotineBSSF
		case "BestLongSideFit":
			return rectpack.GuillotineBLSF
		case "WorstAreaFit":
			return rectpack.GuillotineWAF
		case "WorstShortSideFit":
			return rectpack.GuillotineWSSF
		case "WorstLongSideFit":
			return rectpack.GuillotineWLSF
		}
	case "Skyline":
		switch variant {
		case "BottomLeft":
			return rectpack.SkylineBLF
		case "MinWaste":
			return rectpack.SkylineMinWaste
		}
	}
	panic("invalid algorithm")
}

// OutputResult 输出打包结果
func OutputResult(rectpack *rectpack.Packer) {
	rects := rectpack.Rects()
	size := rectpack.Size()
	fmt.Printf("打包区域大小: %dx%d\n", size.Width, size.Height)
	fmt.Printf("打包效率: %.2f%%\n", rectpack.Used(true)*100)
	fmt.Printf("已打包矩形数量: %d\n", len(rects))
	fmt.Printf("未打包矩形数量: %d\n", len(rectpack.Unpacked()))
	// 输出每个矩形的位置和大小
	fmt.Println("\n矩形详情:")
	for i, rect := range rects {
		fmt.Printf("  矩形 #%d: 位置(%d,%d) 大小(%d,%d)\n",
			i, rect.X, rect.Y, rect.Width, rect.Height)
	}
}

// SaveHTMLVisualization 保存HTML可视化结果
func SaveHTMLVisualization(rectpack *rectpack.Packer, filename string) error {

	// 创建一个临时PNG图像
	pngFilename := "temp_packing.png"
	err := createVisualizationImage(rectpack, pngFilename)
	if err != nil {
		return err
	}

	// 创建HTML文件
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>矩形打包可视化</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .stats { margin-bottom: 20px; }
        .visualization { border: 1px solid #ccc; }
    </style>
</head>
<body>
    <div class="container">
        <h1>矩形打包可视化结果</h1>
        <div class="stats">
            <p><strong>打包区域大小:</strong> {{.Width}}x{{.Height}}</p>
            <p><strong>打包效率:</strong> {{.Efficiency}}%</p>
            <p><strong>已打包矩形数量:</strong> {{.PackedCount}}</p>
            <p><strong>未打包矩形数量:</strong> {{.UnpackedCount}}</p>
        </div>
        <div class="visualization">
            <img src="{{.ImagePath}}" alt="打包可视化">
        </div>
    </div>
</body>
</html>`

	// 准备模板数据
	size := rectpack.Size()
	templateData := struct {
		Width         int
		Height        int
		Efficiency    float64
		PackedCount   int
		UnpackedCount int
		ImagePath     string
	}{
		Width:         size.Width,
		Height:        size.Height,
		Efficiency:    rectpack.Used(true) * 100,
		PackedCount:   len(rectpack.Rects()),
		UnpackedCount: len(rectpack.Unpacked()),
		ImagePath:     pngFilename,
	}

	// 解析并执行模板
	tmpl, err := template.New("visualization").Parse(html)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = tmpl.Execute(file, templateData)
	if err != nil {
		return err
	}

	return nil
}

// createVisualizationImage 创建可视化图像
func createVisualizationImage(rectpack *rectpack.Packer, filename string) error {
	// 获取打包后的矩形和大小
	rects := rectpack.Rects()
	size := rectpack.Size()

	// 创建一个新的RGBA图像
	img := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))

	// 填充黑色背景
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{black}, image.Point{}, draw.Src)

	// 为每个矩形生成随机颜色并绘制
	for _, rect := range rects {
		// 生成随机颜色
		clr := color.RGBA{
			R: uint8(rand.Intn(240) + 15),
			G: uint8(rand.Intn(240) + 15),
			B: uint8(rand.Intn(240) + 15),
			A: 255,
		}

		// 绘制矩形
		r := image.Rect(rect.X, rect.Y, rect.Right(), rect.Bottom())
		draw.Draw(img, r, &image.Uniform{clr}, image.Point{}, draw.Src)

		// 绘制矩形边框
		drawRectBorder(img, rect, color.RGBA{255, 255, 255, 255})
	}

	// 保存为PNG文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// drawRectBorder 绘制矩形边框
func drawRectBorder(img *image.RGBA, rect rectpack.Rect, clr color.RGBA) {
	// 绘制上边框
	for x := rect.X; x < rect.Right(); x++ {
		img.Set(x, rect.Y, clr)
	}

	// 绘制下边框
	for x := rect.X; x < rect.Right(); x++ {
		img.Set(x, rect.Bottom()-1, clr)
	}

	// 绘制左边框
	for y := rect.Y; y < rect.Bottom(); y++ {
		img.Set(rect.X, y, clr)
	}

	// 绘制右边框
	for y := rect.Y; y < rect.Bottom(); y++ {
		img.Set(rect.Right()-1, y, clr)
	}
}
func randomSize(id int, minSize, maxSize rectpack.Size) rectpack.Size {
	w := rand.Intn(maxSize.Width-minSize.Width) + minSize.Width
	h := rand.Intn(maxSize.Height-minSize.Height) + minSize.Height
	return rectpack.NewSizeID(id, w, h)
}

func main() {
	Options := Options{}
	// 定义命令行参数
	Options.AtlasMaxWidth = *flag.Int("width", 2048, "打包区域宽度")
	Options.AtlasMaxHeight = *flag.Int("height", 2048, "打包区域高度")
	Options.IsAllowRotate = *flag.Bool("rotation", false, "允许矩形旋转")
	htmlPtr := flag.String("html", "packing_result.html", "HTML可视化输出文件名")
	// algorithmPtr := flag.String("algorithm", "binarytree", "打包算法 (binarytree, skyline, maxrects, ils)")
	countPtr := flag.Int("count", 30, "矩形数量")
	flag.Parse()

	minSize := rectpack.NewSize(32, 32)
	maxSize := rectpack.NewSize(96, 96)

	// 生成测试用随机尺寸
	sizes := make([]rectpack.Size, *countPtr)
	for i := 0; i < *countPtr; i++ {
		sizes[i] = randomSize(i, minSize, maxSize)
	}

	// 执行打包
	packer, _ := rectpack.NewPacker(Options.AtlasMaxWidth, Options.AtlasMaxHeight, rectpack.MaxRectsBSSF)
	packer.AllowRotate(Options.IsAllowRotate)
	packer.Insert(sizes...)
	packer.Pack()
	// 输出打包结果
	OutputResult(packer)
	// 保存HTML可视化结果
	err := SaveHTMLVisualization(packer, *htmlPtr)
	if err != nil {
		fmt.Printf("保存HTML可视化结果失败: %v\n", err)
	} else {
		fmt.Printf("HTML可视化结果已保存到: %s\n", *htmlPtr)
	}
}
