package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func checkPath(folderPath *string) error {

	fmt.Println("请输入文件夹路径:")
	_, err := fmt.Scan(folderPath)
	if err != nil {
		fmt.Println("读取输入失败:", err)
		return err
	}

	// 检查输入的路径是否是一个有效的文件夹
	info, err := os.Stat(*folderPath)
	if os.IsNotExist(err) {
		fmt.Println("文件夹不存在:", *folderPath)
		return err
	}
	if !info.IsDir() {
		fmt.Println("输入的路径不是一个文件夹:", *folderPath)
		return err
	}

	return nil
}

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
		err := checkPath(&mainFolderPath)
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
