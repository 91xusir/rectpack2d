package rectpack

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// ReadImageFiles 读取目录中的所有图片文件并返回它们的尺寸
func readImageFiles(inputDir string) ([]Size2D, []string, error) {
	// 确保输入目录存在
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("输入目录 %s 不存在", inputDir)
	}

	// 获取所有PNG图片文件
	pattern := filepath.Join(inputDir, "*.png")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("在 %s 目录中没有找到PNG图片", inputDir)
	}
	// 读取每个图片的尺寸
	sizes := make([]Size2D, len(paths))
	for i, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		// 只解码图片头部以获取尺寸信息
		cfg, _, err := image.DecodeConfig(file)
		file.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("无法解码图片 %s: %v", path, err)
		}
		// 创建尺寸对象，使用索引作为ID
		sizes[i] = NewSize2DByID(i, cfg.Width, cfg.Height)

	}

	return sizes, paths, nil
}

// createAtlas 函数用于创建图集，接收一个 Packer 对象和路径列表作为参数
// 它返回一个包含打包图像和路径映射的 RGBA 图像和一个错误（如果有）
func createAtlas(p *Packer, paths []string) (*image.RGBA, map[string]Rect2D, error) {
	// 重置打包器到初始状态
	p.Reset()

	// 遍历每个路径并解码其头部以获取尺寸
	for i, path := range paths {
		var cfg image.Config
		if file, err := os.Open(path); err != nil {
			return nil, nil, err
		} else {
			cfg, _, err = image.DecodeConfig(file)
			file.Close()
			if err != nil {
				return nil, nil, err
			}
		}

		// 将尺寸插入打包器，使用索引作为 ID
		if !p.InsertNewSize2D(i, cfg.Width, cfg.Height) && p.Online {
			// 如果在在线模式下打包，确保每个图像在插入时都能适应
			size := p.MinSize()
			return nil, nil, fmt.Errorf("cannot fit all images into size of %s", size.ToString())
		}
	}

	// 如果在离线模式下打包，执行打包并确保所有图像都已打包
	if !p.Online && !p.Pack() {
		size := p.MinSize()
		return nil, nil, fmt.Errorf("cannot fit all images into size of %s", size.ToString())
	}

	// 获取图集所需的最终尺寸（包括任何配置的填充），并创建一个新图像以进行绘制
	size := p.MinSize()
	mapping := make(map[string]Rect2D, len(paths))
	dst := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	var zero image.Point

	// 遍历每个打包的矩形
	for _, rect := range p.GetPackedRects() {
		// 矩形的 ID 是路径的索引（在上面分配）
		path := paths[rect.ID]
		file, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}

		// 解码该路径上的图像
		src, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			return nil, nil, err
		}

		// 将图像绘制到目标矩形的位置
		bounds := image.Rect(rect.X, rect.Y, rect.Right(), rect.Bottom())
		draw.Draw(dst, bounds, src, zero, draw.Src)

		// 将路径映射到图像绘制的矩形
		mapping[path] = rect
	}

	// 返回结果
	return dst, mapping, nil
}

// randomSize returns a size within the given minimum and maximum sizes.
func randomSize(id int, minSize, maxSize Size2D) Size2D {
	w := rand.Intn(maxSize.Width-minSize.Width) + minSize.Width
	h := rand.Intn(maxSize.Height-minSize.Height) + minSize.Height
	return NewSize2DByID(id, w, h)
}

// randomColor (surprise!) returns a random color.
func randomColor() color.RGBA {
	// Offset to use a minimum value so it is never pure black.
	return color.RGBA{
		R: uint8(rand.Intn(240)) + 15,
		G: uint8(rand.Intn(240)) + 15,
		B: uint8(rand.Intn(240)) + 15,
		A: 255,
	}
}

// createImage colorizes and creates an image from packed rectangles to provide
// a visual representation.
func createImage(t *testing.T, path string, packer *Packer) {
	black := color.RGBA{0, 0, 0, 255}
	size := packer.MinSize()
	img := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{black}, image.Point{}, draw.Src)

	for _, rect := range packer.GetPackedRects() {
		color := randomColor()
		r := image.Rect(rect.X, rect.Y, rect.Right(), rect.Bottom())
		draw.Draw(img, r, &image.Uniform{color}, image.Point{0, 0}, draw.Src)
	}

	if file, err := os.Create(path); err == nil {
		defer file.Close()
		png.Encode(file, img)
	} else {
		t.Fatal(err)
	}
}

func TestAtlas(t *testing.T) {
	paths, _ := filepath.Glob("../demo/*.png")
	packer, _ := NewPacker(5120, 5120, MaxRectsBAF)

	img, mapping, err := createAtlas(packer, paths)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	for k, v := range mapping {
		fmt.Printf("%v: %s\n", k, v.String())
	}

	file, _ := os.Create("atlas.png")
	defer file.Close()
	png.Encode(file, img)
}
func TestRandom(t *testing.T) {
	const (
		count       = 1024
		atlasWidth  = 1024
		atlasHeight = 1024
	)
	minSize := NewSize2D(32, 32)
	maxSize := NewSize2D(96, 96)

	// 生成测试用随机尺寸
	sizes := make([]Size2D, count)
	for i := 0; i < count; i++ {
		sizes[i] = randomSize(i, minSize, maxSize)
	}

	// 所有待打包尺寸
	unpacked := slices.Clone(sizes)
	atlases := []*Packer{}
	atlasIndex := 0

	for len(unpacked) > 0 {
		packer, _ := NewPacker(atlasWidth, atlasHeight, MaxRectsBSSF)
		packer.AllowRotate(true)
		packer.Online = false
		packer.padding = 2
		packer.SetSorter(SortArea, false)
		packer.Insert(unpacked...)

		ok := packer.Pack()
		atlases = append(atlases, packer)

		if !ok {
			// 打包失败，保留未打包部分继续处理
			unpacked = slices.Clone(packer.GetUnpackedRects())
		} else {
			unpacked = nil
		}

		atlasIndex++
		if atlasIndex > 10 {
			t.Fatal("Too many atlases required; check for very large sizes")
		}
	}

	// 验证图块没有重叠
	for idx, atlas := range atlases {
		rects := atlas.GetPackedRects()
		for i := 0; i < len(rects)-1; i++ {
			for j := i + 1; j < len(rects); j++ {
				if rects[i].Intersects(rects[j]) {
					t.Errorf("Atlas %d: %s and %s intersect", idx, rects[i].String(), rects[j].String())
				}
			}
		}
		createImage(t, fmt.Sprintf("packed_%d.png", idx), atlas)
	}
}

func runBenchmark(algoName, variant string) {
	//时间统计
	start := time.Now() // 记录开始时间
	heuristic := ResolveAlgorithm(algoName, variant)
	sizes, _, _ := readImageFiles("../input")
	packer, _ := NewPacker(5120, 5120, heuristic)
	packer.Insert(sizes...)
	packer.Pack()
	packer.Shrink()
	elapsed := time.Since(start) // 计算耗时
	fmt.Printf("%s-%s | 利用率:%.2f%% | 用时: %s\n", algoName, variant, packer.GetAreaUsedRate(true)*100, elapsed)
}

func Test_alog(t *testing.T) {
	algos := map[string][]string{
		"MaxRects":   {"BestShortSideFit", "BottomLeft", "ContactPoint", "BestLongSideFit", "BestAreaFit"},
		"Guillotine": {"BestAreaFit", "BestShortSideFit", "BestLongSideFit", "WorstAreaFit", "WorstShortSideFit", "WorstLongSideFit"},
		"Skyline":    {"BottomLeft", "MinWaste"},
	}
	for algo, variants := range algos {
		for _, variant := range variants {
			t.Run(algo+"_"+variant, func(t *testing.T) {
				runBenchmark(algo, variant)
			})
		}
	}
}
