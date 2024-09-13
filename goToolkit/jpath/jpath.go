package jpath

import (
	"fmt"
	"os"
)

func InputFolderAndCheck(folderPath *string) error {

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
