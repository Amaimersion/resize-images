package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
)

func main() {
	var root string
	var dest string
	var maxGoroutines int

	flag.StringVar(
		&root,
		"root",
		"",
		"Root",
	)
	flag.StringVar(
		&dest,
		"dest",
		"",
		"Dest",
	)
	flag.IntVar(
		&maxGoroutines,
		"threads",
		10,
		"Max threads",
	)
	flag.Parse()

	if len(root) == 0 || len(dest) == 0 {
		os.Exit(1)
	}

	var wg sync.WaitGroup
	guard := make(chan struct{}, maxGoroutines)

	fileSystem := os.DirFS(root)
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

			imgPath := filepath.Join(root, p)
			img, err := imaging.Open(imgPath)

			if err != nil {
				fmt.Printf("%s: %s\n", imgPath, err)
				return
			}

			newImgPath := filepath.Join(dest, p)
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
