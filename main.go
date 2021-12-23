package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
)

func main() {
	cfg := parseFlags()

	if err := cfg.isValid(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	workers := make(chan struct{}, cfg.maxThreads)

	fsys := os.DirFS(cfg.sourcePath)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		workers <- struct{}{}
		wg.Add(1)

		go func() {
			defer func() {
				<-workers
				wg.Done()
			}()

			args := resizeImageArgs{
				sourcePath: cfg.sourcePath,
				destPath:   cfg.destPath,
				imageName:  path,
				maxWidth:   cfg.maxWidth,
				jpgQuality: cfg.jpgQuality,
			}

			if err := resizeImage(args); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()

		return nil
	})

	wg.Wait()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type config struct {
	sourcePath string
	destPath   string
	maxThreads int
	maxWidth   int
	jpgQuality int
}

func (c config) isValid() error {
	if len(c.sourcePath) == 0 {
		return errors.New("source path must be specified")
	}

	if len(c.destPath) == 0 {
		return errors.New("destination path must be specified")
	}

	return nil
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
		"Result image width will be not more than this value.",
	)
	flag.IntVar(
		&cfg.jpgQuality,
		"quality",
		100,
		"Result image will have this quality. From 0 to 100. Only applied to JPG images.",
	)

	flag.Parse()

	return cfg
}

type resizeImageArgs struct {
	sourcePath string
	destPath   string
	imageName  string
	maxWidth   int
	jpgQuality int
}

func resizeImage(args resizeImageArgs) error {
	imgPath := filepath.Join(args.sourcePath, args.imageName)
	img, err := imaging.Open(imgPath)

	if err != nil {
		return fmt.Errorf("error: %v: %v", args.imageName, err)
	}

	resized := false

	if width := img.Bounds().Max.X; width > args.maxWidth {
		img = imaging.Resize(img, args.maxWidth, 0, imaging.Lanczos)
		resized = true
	}

	newImgPath := filepath.Join(args.destPath, args.imageName)
	newImgDir := strings.TrimSuffix(newImgPath, filepath.Base(newImgPath))

	if err := os.MkdirAll(newImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("error: %v: %v", args.imageName, err)
	}

	if err := imaging.Save(img, newImgPath, imaging.JPEGQuality(args.jpgQuality)); err != nil {
		return fmt.Errorf("error: %v: %v", args.imageName, err)
	}

	if resized {
		fmt.Printf("resized: %v\n", args.imageName)
	} else {
		fmt.Printf("not resized: %v\n", args.imageName)
	}

	return nil
}
