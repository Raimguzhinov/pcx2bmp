package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/veandco/go-sdl2/sdl"
)

type Quantizer interface {
	Quantize(*ImageData) (*ImageData, error)
}

type Options struct {
	Output  string `short:"o" long:"output" description:"Имя выходного BMP-файла"`
	Show    bool   `short:"s" long:"show" description:"Отобразить изображение после конвертации"`
	Version bool   `short:"v" long:"version" description:"Показать версию и выйти"`
	Help    bool   `short:"h" long:"help" description:"Показать справку с описанием алгоритма"`
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.IgnoreUnknown)
	args, err := parser.Parse()
	if opts.Help {
		fmt.Print(detailedHelp)
		return
	}
	if opts.Version {
		fmt.Println(version)
		return
	}
	if err != nil || len(args) == 0 {
		fmt.Print(detailedHelp)
		os.Exit(1)
	}

	inputFile := args[0]
	outputFile := opts.Output

	// Если `--output` не указан, используем `<input>.bmp`
	if outputFile == "" {
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".bmp"
	}

	pcxDataCh := make(chan *ImageData)
	quantDataCh := make(chan *ImageData)
	errCh := make(chan error)
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		imgData, err := LoadPCX(inputFile)
		if err != nil {
			errCh <- err
			return
		}
		pcxDataCh <- imgData
		close(pcxDataCh)
	}()

	go func() {
		defer wg.Done()
		var q Quantizer = &FreqQuantizer{}
		for imgData := range pcxDataCh {
			newImg, err := q.Quantize(imgData)
			if err != nil {
				errCh <- err
				return
			}
			quantDataCh <- newImg
		}
		close(quantDataCh)
	}()

	go func() {
		defer wg.Done()
		for qImg := range quantDataCh {
			if err := SaveBMP(outputFile, qImg); err != nil {
				errCh <- err
				return
			}
			done <- struct{}{}
		}
	}()

	select {
	case err := <-errCh:
		log.Fatalf("Ошибка в конвейере: %v", err)
	case <-done:
		log.Println("Файл успешно записан:", filepath.Join(".", outputFile))
	}

	wg.Wait()

	if opts.Show {
		if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
			log.Fatalf("Ошибка инициализации SDL: %v", err)
		}
		defer sdl.Quit()

		original, err := LoadPCX(inputFile)
		if err != nil {
			log.Fatalf("Ошибка загрузки PCX: %v", err)
		}
		converted, err := LoadBMP(outputFile)
		if err != nil {
			log.Fatalf("Ошибка загрузки BMP: %v", err)
		}

		winOrig, rendOrig, texOrig, err := createWindowAndTexture("Original PCX", original, 100, 100)
		if err != nil {
			log.Fatal(err)
		}
		defer winOrig.Destroy()
		defer rendOrig.Destroy()
		defer texOrig.Destroy()

		winConv, rendConv, texConv, err := createWindowAndTexture("Converted to 256 BMP", converted, 300, 150)
		if err != nil {
			log.Fatal(err)
		}
		defer winConv.Destroy()
		defer rendConv.Destroy()
		defer texConv.Destroy()

		showLoop(winOrig, rendOrig, texOrig, winConv, rendConv, texConv)
	}
}

func showLoop(winOrig *sdl.Window, rendOrig *sdl.Renderer, texOrig *sdl.Texture,
	winConv *sdl.Window, rendConv *sdl.Renderer, texConv *sdl.Texture,
) {
	quitCh := make(chan struct{})
	waitClose(quitCh, func(windowId uint32) {
		if origId, _ := winOrig.GetID(); windowId == origId {
			log.Println("Окно Original PCX закрыто")
			winOrig.Destroy()
		}
		if convId, _ := winConv.GetID(); windowId == convId {
			log.Println("Окно Converted закрыто")
			winConv.Destroy()
		}
	})

	for {
		select {
		case <-quitCh:
			log.Println("Завершение SDL-цикла")
			return
		default:
			renderWindow(winOrig, rendOrig, texOrig)
			renderWindow(winConv, rendConv, texConv)
			sdl.Delay(16) // ~60 FPS
		}
	}
}

func renderWindow(win *sdl.Window, rend *sdl.Renderer, tex *sdl.Texture) {
	if win == nil || rend == nil || tex == nil {
		return
	}
	rend.SetDrawColor(0, 0, 0, 255)
	rend.Clear()
	rend.Copy(tex, nil, nil)
	rend.Present()
}

func createWindowAndTexture(title string, img *ImageData, x, y int) (*sdl.Window, *sdl.Renderer, *sdl.Texture, error) {
	w, h := img.Width, img.Height

	win, err := sdl.CreateWindow(title, int32(x), int32(y), int32(w), int32(h), sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, nil, nil, err
	}
	rend, err := sdl.CreateRenderer(win, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		win.Destroy()
		return nil, nil, nil, err
	}
	tex, err := rend.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, int32(w), int32(h))
	if err != nil {
		rend.Destroy()
		win.Destroy()
		return nil, nil, nil, err
	}

	rawRGBA := make([]byte, w*h*4)
	idx := 0
	for i := 0; i < len(rawRGBA); i += 4 {
		c := img.Pix[idx]
		rawRGBA[i+0] = c.R
		rawRGBA[i+1] = c.G
		rawRGBA[i+2] = c.B
		rawRGBA[i+3] = 255
		idx++
	}
	pixels, pitch, err := tex.Lock(nil)
	if err != nil {
		tex.Destroy()
		rend.Destroy()
		win.Destroy()
		return nil, nil, nil, err
	}
	for row := 0; row < h; row++ {
		startSrc := row * w * 4
		endSrc := startSrc + w*4
		startDst := row * pitch
		copy(pixels[startDst:startDst+w*4], rawRGBA[startSrc:endSrc])
	}
	tex.Unlock()
	return win, rend, tex, nil
}

func waitClose(quitCh chan struct{}, closeWindowCb func(windowId uint32)) {
	go func() {
		for {
			select {
			case <-quitCh:
				return
			default:
				ev := sdl.PollEvent()
				if ev == nil {
					sdl.Delay(10)
					continue
				}
				switch e := ev.(type) {
				case *sdl.QuitEvent:
					close(quitCh)
					return
				case *sdl.WindowEvent:
					if e.Event == sdl.WINDOWEVENT_CLOSE {
						closeWindowCb(e.WindowID)
						close(quitCh)
						return
					}
				}
			}
		}
	}()
}
