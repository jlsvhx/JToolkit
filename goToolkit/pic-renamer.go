package main

import (
	"fmt"
	"goToolkit/jpath"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func listSubFolder(path string) []string {

	var subFolderList []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			//fmt.Println(path)
			subFolderList = append(subFolderList, path)
		}
		return nil
	})
	if err != nil {

	}

	return subFolderList
}

func renameFiles(srcPath string) error {
	defer func() {
		fmt.Println(srcPath, "文件夹处理完成")
	}()
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	namePrefix := info.Name()
	ymd := time.Now().Format("20060102") // 使用 Go 的时间格式化
	count := 1
	list, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, item := range list {
		itemPath := filepath.Join(srcPath, item.Name())
		info, err := os.Stat(itemPath)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileName := info.Name()
			ext := filepath.Ext(fileName)
			newFileName := fmt.Sprintf("%s-%s-%d%s", namePrefix, ymd, count, ext)
			tmpFileName := filepath.Join(srcPath, newFileName)
			if !strings.HasPrefix(fileName, namePrefix) {
				fmt.Println(fileName, "=>", newFileName)
				err := os.Rename(itemPath, tmpFileName)
				if err != nil {
					fmt.Println(err)
				}
				count++
			}
		}
	}

	return nil
}

func main() {
	var mainFolderPath string
	for true {
		err := jpath.InputFolderAndCheck(&mainFolderPath)
		if err == nil {
			break
		}
	}
	subFolderList := listSubFolder(mainFolderPath)
	fmt.Println(subFolderList)
	var wg sync.WaitGroup
	for _, folder := range subFolderList {
		wg.Add(1)
		go func(folder string) {
			defer wg.Done()
			err := renameFiles(folder)
			if err != nil {
				fmt.Println(err)
			}
		}(folder)
	}
	wg.Wait()
	_, err := fmt.Scanln()
	if err != nil {
		return
	}
}
