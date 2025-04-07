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
)

// createAtlas 函数用于创建图集，接收一个 Packer 对象和路径列表作为参数
// 它返回一个包含打包图像和路径映射的 RGBA 图像和一个错误（如果有）
func createAtlas(p *Packer, paths []string) (*image.RGBA, map[string]Rect, error) {
	// 重置打包器到初始状态
	p.Clear()

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
		if !p.InsertSize(i, cfg.Width, cfg.Height) && p.Online {
			// 如果在在线模式下打包，确保每个图像在插入时都能适应
			size := p.Size()
			return nil, nil, fmt.Errorf("cannot fit all images into size of %s", size.String())
		}
	}

	// 如果在离线模式下打包，执行打包并确保所有图像都已打包
	if !p.Online && !p.Pack() {
		size := p.Size()
		return nil, nil, fmt.Errorf("cannot fit all images into size of %s", size.String())
	}

	// 获取图集所需的最终尺寸（包括任何配置的填充），并创建一个新图像以进行绘制
	size := p.Size()
	mapping := make(map[string]Rect, len(paths))
	dst := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	var zero image.Point

	// 遍历每个打包的矩形
	for _, rect := range p.Rects() {
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
func randomSize(id int, minSize, maxSize Size) Size {
	w := rand.Intn(maxSize.Width-minSize.Width) + minSize.Width
	h := rand.Intn(maxSize.Height-minSize.Height) + minSize.Height
	return NewSizeID(id, w, h)
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
	size := packer.Size()
	img := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{black}, image.Point{}, draw.Src)

	for _, rect := range packer.Rects() {
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
	return
	paths, _ := filepath.Glob("/usr/share/icons/Adwaita/32x32/devices/*.png")
	packer, _ := NewPacker(512, 512, MaxRectsBAF)

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
	minSize := NewSize(32, 32)
	maxSize := NewSize(96, 96)

	// 生成测试用随机尺寸
	sizes := make([]Size, count)
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
		packer.Padding = 2
		packer.Sorter(SortArea, false)
		packer.Insert(unpacked...)

		ok := packer.Pack()
		atlases = append(atlases, packer)

		if !ok {
			// 打包失败，保留未打包部分继续处理
			unpacked = slices.Clone(packer.Unpacked())
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
		rects := atlas.Rects()
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
