# pcx2bmp — Convert PCX->BMP 

## Requirements
* [Go v1.20+](https://go.dev/dl/)
* [SDL2](https://github.com/libsdl-org/SDL/releases)
* [SDL2_image (optional)](https://github.com/libsdl-org/SDL_image/releases)
* [SDL2_mixer (optional)](https://github.com/libsdl-org/SDL_mixer/releases)
* [SDL2_ttf (optional)](https://github.com/libsdl-org/SDL_ttf/releases)
* [SDL2_gfx (optional)](http://www.ferzkopp.net/wordpress/2016/01/02/sdl_gfx-sdl2_gfx/)

On __Ubuntu 22.04 and above__, type:\
`apt install libsdl2{,-image,-mixer,-ttf,-gfx}-dev`

On __Fedora 36 and above__, type:\
`dnf install SDL2{,_image,_mixer,_ttf,_gfx}-devel`

On __Arch Linux__, type:\
`pacman -S sdl2{,_image,_mixer,_ttf,_gfx}`

On __Gentoo__, type:\
`emerge -av libsdl2 sdl2-{image,mixer,ttf,gfx}`

On __macOS__, install SDL2 via [Homebrew](http://brew.sh) like so:\
`brew install sdl2{,_image,_mixer,_ttf,_gfx} pkg-config`

On __Windows__,
1. Install mingw-w64 from [Mingw-builds](https://github.com/niXman/mingw-builds-binaries/releases). A 7z archive extractor software might be needed which can be downloaded [here](https://www.7-zip.org/download.html). In this example, we extract the content, which is `mingw64`, into `C:\`.
2. Download and install `SDL2-devel-[version]-mingw.zip` files from https://github.com/libsdl-org/SDL/releases.
    * Extract the SDL2 folder from the archive using a tool like [7zip](http://7-zip.org)
    * Inside the extracted SDL2 folder, copy the `i686-w64-mingw32` and/or `x86_64-w64-mingw32` into mingw64 folder e.g. `C:\mingw64`
3. Setup `Path` environment variable
    * Put mingw-w64 binaries location into system `Path` environment variable (e.g. `C:\mingw64\bin`)
4. Close and open terminal again so the new `Path` environment variable takes effect. Now we should be able to run `go build` inside the project directory.
5. Download and install SDL2 runtime libraries from https://github.com/libsdl-org/SDL/releases. Extract and copy the `.dll` file into the project directory. After that, the program should become runnable.
6. (Optional) You can repeat __Step 2__ for [SDL_image](https://github.com/libsdl-org/SDL_image/releases), [SDL_mixer](https://github.com/libsdl-org/SDL_mixer/releases), [SDL_ttf](https://github.com/libsdl-org/SDL_ttf/releases)

## Installation
```bash
go install github.com/Raimguzhinov/pcx2bmp@latest
```

## Usage

```
pcx2bmp <входной файл> [опции]

ОПЦИИ:
----------------------------------------------------------------------
  -o, --output <файл>  Имя выходного BMP-файла (по умолчанию <input>.bmp)
  -s, --show           Отобразить изображение после конвертации
  -v, --version        Показать версию программы
  -h, --help           Показать эту справку

ПРИМЕРЫ:
----------------------------------------------------------------------
  pcx2bmp image.pcx               # Конвертировать в image.bmp
  pcx2bmp image.pcx -o result.bmp # Указать имя выходного файла
  pcx2bmp image.pcx --show        # Конвертировать и показать изображение
```
