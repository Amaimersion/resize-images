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
		"Result image width will be no more than this value.",
	)

	flag.Parse()

	return cfg
}

type resizeImageArgs struct {
	sourcePath string
	destPath   string
	imageName  string
	maxWidth   int
}

func resizeImage(args resizeImageArgs) error {
	imgPath := filepath.Join(args.sourcePath, args.imageName)
	img, err := imaging.Open(imgPath)

	if err != nil {
		return err
	}

	newImgPath := filepath.Join(args.destPath, args.imageName)
	newImgDir := strings.TrimSuffix(newImgPath, filepath.Base(newImgPath))

	if err := os.MkdirAll(newImgDir, os.ModePerm); err != nil {
		return err
	}

	if width := img.Bounds().Max.X; width > args.maxWidth {
		newImg := imaging.Resize(img, args.maxWidth, 0, imaging.Lanczos)

		if err := imaging.Save(newImg, newImgPath); err != nil {
			return err
		}

		fmt.Printf("Resized: %v\n", imgPath)
	} else {
		if err := imaging.Save(img, newImgPath); err != nil {
			return err
		}

		fmt.Printf("Not modified: %v\n", imgPath)
	}

	return nil
}
