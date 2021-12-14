package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
)

func main() {
	cfg := parseFlags()

	if len(cfg.sourcePath) == 0 || len(cfg.destPath) == 0 {
		os.Exit(1)
	}

	var wg sync.WaitGroup
	guard := make(chan struct{}, cfg.maxThreads)

	fileSystem := os.DirFS(cfg.sourcePath)
	err := fs.WalkDir(fileSystem, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		guard <- struct{}{}

		wg.Add(1)
		go func() {
			defer func() {
				<-guard
				wg.Done()
			}()

			imgPath := filepath.Join(cfg.sourcePath, p)
			img, err := imaging.Open(imgPath)

			if err != nil {
				fmt.Printf("%s: %s\n", imgPath, err)
				return
			}

			newImgPath := filepath.Join(cfg.destPath, p)
			newImgDir := strings.TrimSuffix(newImgPath, filepath.Base(newImgPath))

			if err := os.MkdirAll(newImgDir, os.ModePerm); err != nil {
				fmt.Printf("%s: %s\n", newImgPath, err)
				return
			}

			if width := img.Bounds().Max.X; width > 1600 {
				newImg := imaging.Resize(img, 1600, 0, imaging.Lanczos)

				if err := imaging.Save(newImg, newImgPath); err == nil {
					fmt.Printf("Resized: %v\n", imgPath)
				} else {
					fmt.Printf("%s: %s\n", newImgPath, err)
				}
			} else {
				if err := imaging.Save(img, newImgPath); err == nil {
					fmt.Printf("Not modified: %v\n", imgPath)
				} else {
					fmt.Printf("%s: %s\n", newImgPath, err)
				}
			}
		}()

		return nil
	})

	wg.Wait()

	if err != nil {
		log.Fatal(err)
	}
}

type config struct {
	sourcePath string
	destPath   string
	maxThreads int
	maxWidth   int
}

func parseFlags() config {
	cfg := config{}

	flag.StringVar(
		&cfg.sourcePath,
		"source",
		"",
		"Path to source folder.",
	)
	flag.StringVar(
		&cfg.destPath,
		"dest",
		"",
		"Path to destination folder.",
	)
	flag.IntVar(
		&cfg.maxThreads,
		"threads",
		runtime.NumCPU(),
		"Maximum number of images that will be processed concurrently.",
	)
	flag.IntVar(
		&cfg.maxWidth,
		"width",
		1920,
		"Result image width will be no more than this value.",
	)

	flag.Parse()

	return cfg
}
