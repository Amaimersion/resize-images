# resize-images

This program is intended to decrease storage size of folder of images. This is achieved by decreasing images size and images quality. Result will be a separate folder with modified images.

## How to run

Download and execute binary file through terminal. Use `--help` flag to see help message.

## Notes

- Original files will be not touched.
- Original folder structure will be keeped.
- Original file names will be keeped.
- Non-images will be not moved in result folder.

## Example

```shell
./resize-images --source /home/sergey/files --dest /home/sergey/resized-images --width 1920 --quality 95 --threads 4
```
