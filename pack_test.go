package main

import (
	"bufio"
	"fmt"
	"os"
	"rectpack2d/rectpack"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func printElapsed(t time.Duration) {
	switch {
	case t < time.Microsecond:
		fmt.Printf("Time used: %d ns\n", t.Nanoseconds())
	case t < time.Millisecond:
		fmt.Printf("Time used: %.2f µs\n", float64(t.Microseconds()))
	case t < time.Second:
		fmt.Printf("Time used: %.2f ms\n", float64(t.Milliseconds()))
	default:
		fmt.Printf("Time used: %.2f s\n", t.Seconds())
	}
}

func Test_maxrects(t *testing.T) {
	path := "data.txt"
	box, err := GetInstance(path)
	if err != nil {
		fmt.Printf("read file error: %v\n", err)
		return
	}
	size2Ds := make([]rectpack.Size2D, len(box.ReqPackRectList))
	for i := range box.ReqPackRectList {
		size2Ds = append(size2Ds, rectpack.NewSize2DByID(box.ReqPackRectList[i].w, box.ReqPackRectList[i].h, box.ReqPackRectList[i].id))
	}
	sort.Slice(size2Ds, func(i, j int) bool {
		return size2Ds[i].Width*size2Ds[i].Height < size2Ds[j].Width*size2Ds[j].Height
	})

	packer, _ := rectpack.NewPacker(400, 400, rectpack.MaxRectsBSSF)
	packer.Insert(size2Ds...)
	startTime := time.Now()

	packer.Pack()
	elapsedTime := time.Since(startTime)
	printElapsed(elapsedTime)

	rects := packer.GetPackedRects()
	r := make([]*PackedRect, len(rects))
	for i := range rects {
		r[i] = &PackedRect{}
		r[i].id = rects[i].ID
		r[i].x = rects[i].X
		r[i].y = rects[i].Y
		r[i].w = rects[i].Width
		r[i].h = rects[i].Height
		r[i].isRotated = false
	}
	fmt.Printf("packed rectangles: %d\n", len(packer.GetPackedRects()))
	fmt.Printf("utilization rate: %f\n", packer.GetAreaUsedRate(false))
	generateVisualizationHTML(r, "MaxRect_Algorithm_Visualization")

	// packer.Shrink()
}

func Test_skyline(t *testing.T) {
	// Data address

	path := "data.txt"
	// Get instance object based on txt file

	box, err := GetInstance(path)
	if err != nil {
		fmt.Printf("read file error: %v\n", err)
		return
	}
	// Record the algorithm start time

	startTime := time.Now()
	// Get rectangle slices

	rects := box.ReqPackRectList
	// Arrange in descending order of area

	sort.Slice(rects, func(i, j int) bool {
		return rects[i].w*rects[i].h > rects[j].w*rects[j].h
	})

	// Instantiate the skyline object

	skyLinePacking := NewSkyLinePacking(box.IsRotateEnable, box.W, box.H, rects)
	// Call the skyline algorithm for solving

	solution, err := skyLinePacking.Pack()
	if err != nil {
		fmt.Printf("skyline algorithm error: %v\n", err)
		return
	}
	// Output related information

	elapsedTime := time.Since(startTime)
	printElapsed(elapsedTime)
	fmt.Printf("packed rectangles: %d\n", len(solution.packedRectList))
	fmt.Printf("utilization rate: %f\n", solution.rate)

	// 使用提取的函数生成可视化HTML
	generateVisualizationHTML(solution.packedRectList, "Skyline_Algorithm_Visualization")
}

// generateVisualizationHTML 生成矩形排列的可视化HTML文件
func generateVisualizationHTML(packedRects []*PackedRect, title string) {
	strings1 := make([]string, len(packedRects))
	strings2 := make([]string, len(packedRects))
	for i, rect := range packedRects {
		strings1[i] = fmt.Sprintf("{x:%v,y:%v,l:%v,w:%v}", rect.x, rect.y, rect.h, rect.w)
		if rect.isRotated {
			strings2[i] = "1"
		} else {
			strings2[i] = "0"
		}
	}
	fmt.Printf("data: %v,\n", strings1)
	fmt.Printf("isRotate: %v,\n", strings2)
	// Stitching HTML strings

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>%s</title>
  <style>
    canvas { border: 1px solid #ccc; background: #fff; }
  </style>
</head>
<body>
  <h3>%s</h3>
  <canvas id="canvas" width="800" height="800"></canvas>
  <script>
    const data = [%v];
    const isRotate = [%v];
    const canvas = document.getElementById("canvas");
    const ctx = canvas.getContext("2d");
    let maxX = 0, maxY = 0;
    data.forEach(rect => {
      const x2 = rect.x + rect.w;
      const y2 = rect.y + rect.l;
      if (x2 > maxX) maxX = x2;
      if (y2 > maxY) maxY = y2;
    });
    const scale = Math.min(canvas.width / maxX, canvas.height / maxY);
    data.forEach((rect, i) => {
      const color = "#" + Math.floor(Math.random()*16777215).toString(16).padStart(6, "0");
      const x = rect.x * scale;
      const y = rect.y * scale;
      const w = rect.w * scale;
      const h = rect.l * scale;
      ctx.fillStyle = color;
      ctx.fillRect(x, y, w, h);
      ctx.strokeStyle = "black";
      ctx.strokeRect(x, y, w, h);
      ctx.fillStyle = "black";
      ctx.font = "12px Arial";
      ctx.fillText(i + (isRotate[i] === 1 ? " (R)" : ""), x + 3, y + 12);
    });
  </script>
</body>
</html>
`, title, title, strings.Join(strings1, ","), strings.Join(strings2, ","))

	err := os.WriteFile(title+".html", []byte(html), 0644)
	if err != nil {
		fmt.Println("write file error:", err)
	} else {
		fmt.Println("✅ Visual HTML file generated: output.html")
	}
}

func GetInstance(path string) (*Bin, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	box := &Bin{}
	var rectList []*Rect
	isFirstLine := true
	id := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if isFirstLine {
			w, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("an error in parsing width: %w", err)
			}
			h, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("an error in parsing height: %w", err)
			}
			rotateEnable := parts[2] == "1"

			box.W = w
			box.H = h
			box.IsRotateEnable = rotateEnable
			isFirstLine = false
		} else {
			w, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("an error in parsing rectangle width: %w", err)
			}
			h, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("an error in parsing rectangle height: %w", err)
			}
			rect := NewRect(id, w, h)
			id++
			rectList = append(rectList, rect)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	box.ReqPackRectList = rectList
	return box, nil
}
