package main

import (
	"math"
	"runtime"
	"sort"
	"sync"
)

type FreqQuantizer struct{}

func (q *FreqQuantizer) Quantize(img *ImageData) (*ImageData, error) {
	freqMap := make(map[ColorRGB]int)

	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	var mu sync.Mutex

	step := len(img.Pix) / numCPU
	for i := 0; i < numCPU; i++ {
		start := i * step
		end := start + step
		if i == numCPU-1 {
			end = len(img.Pix)
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			localMap := make(map[ColorRGB]int)
			for j := s; j < e; j++ {
				c := img.Pix[j]
				localMap[c]++
			}
			mu.Lock()
			for k, v := range localMap {
				freqMap[k] += v
			}
			mu.Unlock()
		}(start, end)
	}
	wg.Wait()

	type kv struct {
		c ColorRGB
		n int
	}
	arr := make([]kv, 0, len(freqMap))
	for c, n := range freqMap {
		arr = append(arr, kv{c, n})
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].n > arr[j].n
	})

	var palette [256]ColorRGB
	for i := 0; i < 256; i++ {
		if i < len(arr) {
			palette[i] = arr[i].c
		} else {
			palette[i] = ColorRGB{0, 0, 0}
		}
	}

	outPix := make([]ColorRGB, len(img.Pix))
	step = len(img.Pix) / numCPU
	var wg2 sync.WaitGroup
	for i := 0; i < numCPU; i++ {
		start := i * step
		end := start + step
		if i == numCPU-1 {
			end = len(img.Pix)
		}
		wg2.Add(1)
		go func(s, e int) {
			defer wg2.Done()
			for j := s; j < e; j++ {
				c := img.Pix[j]
				idx := findNearestIndex(c, palette)
				outPix[j] = ColorRGB{R: byte(idx), G: 0, B: 0}
			}
		}(start, end)
	}
	wg2.Wait()

	res := &ImageData{Width: img.Width, Height: img.Height, Pix: outPix, Palette: palette, HasPal: true}
	return res, nil
}

func findNearestIndex(c ColorRGB, palette [256]ColorRGB) int {
	best := 0
	bestDist := math.MaxFloat64
	for i := 0; i < 256; i++ {
		d := distSq(c, palette[i])
		if d < bestDist {
			bestDist = d
			best = i
			if bestDist == 0 {
				break
			}
		}
	}
	return best
}

func distSq(a, b ColorRGB) float64 {
	dr := float64(int(a.R) - int(b.R))
	dg := float64(int(a.G) - int(b.G))
	db := float64(int(a.B) - int(b.B))
	return dr*dr + dg*dg + db*db
}
