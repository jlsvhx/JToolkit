package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/md5"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const dbName = "file_integrity_check.db"

var excludeFileNames = []string{dbName}
var dbLock sync.Mutex

func calculateMD5(filePath string) (string, error) {
	var h hash.Hash = md5.New()
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func getFileModificationTime(filePath string) (time.Time, error) {
	return os.Stat(filePath).ModTime(), nil
}

func initializeDB(dbPath string) error {
	dbLock.Lock()
	defer dbLock.Unlock()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files_md5 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		md5 TEXT NOT NULL,
		modification_time TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_file_path ON files_md5 (file_path)`)
	return err
}

func pushFilesToProcess(mainDirectory string, taskQueue chan string) {
	filepath.Walk(mainDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		for _, exclude := range excludeFileNames {
			if info.Name() == exclude || filepath.Ext(info.Name()) == ".sfv" {
				return nil
			}
		}
		taskQueue <- path
		return nil
	})
}

func verifyMD5InFolder(mainDirectory string, threadCount int) {
	taskQueue := make(chan string, 100)
	resultQueue := make(chan string)

	dbPath := filepath.Join(mainDirectory, dbName)
	if err := initializeDB(dbPath); err != nil {
		fmt.Println("数据库初始化失败:", err)
		return
	}

	go pushFilesToProcess(mainDirectory, taskQueue)

	var wg sync.WaitGroup
	totalFiles := 0

	go func() {
		for range taskQueue {
			totalFiles++
		}
	}()

	bar := progressbar.NewOptions(totalFiles, progressbar.OptionSetDescription("Processing files"))

	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range taskQueue {
				currentMD5, err := calculateMD5(filePath)
				if err != nil {
					resultQueue <- fmt.Sprintf("文件 %s 计算MD5失败: %v", filePath, err)
					continue
				}
				currentModTime, err := getFileModificationTime(filePath)
				if err != nil {
					resultQueue <- fmt.Sprintf("文件 %s 获取修改时间失败: %v", filePath, err)
					continue
				}
				relativePath, _ := filepath.Rel(mainDirectory, filePath)

				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					resultQueue <- fmt.Sprintf("数据库连接失败: %v", err)
					continue
				}
				defer db.Close()

				var dbMD5 string
				var dbModTimeStr string
				err = db.QueryRow(`SELECT md5, modification_time FROM files_md5 WHERE file_path = ?`, relativePath).Scan(&dbMD5, &dbModTimeStr)

				if err == sql.ErrNoRows {
					resultQueue <- fmt.Sprintf("未找到对应的MD5记录，已添加: %s", filePath)
					dbLock.Lock()
					_, _ = db.Exec(`INSERT INTO files_md5 (file_path, md5, modification_time) VALUES (?, ?, ?)`,
						relativePath, currentMD5, currentModTime.Format(time.RFC3339))
					dbLock.Unlock()
				} else if err != nil {
					resultQueue <- fmt.Sprintf("查询数据库失败: %v", err)
				} else {
					dbModTime, _ := time.Parse(time.RFC3339, dbModTimeStr)

					if currentModTime == dbModTime {
						if currentMD5 == dbMD5 {
							resultQueue <- fmt.Sprintf("MD5与数据库一致: %s", filePath)
						} else {
							resultQueue <- fmt.Sprintf("文件损坏: %s", filePath)
						}
					} else {
						dbLock.Lock()
						_, _ = db.Exec(`UPDATE files_md5 SET md5 = ?, modification_time = ? WHERE file_path = ?`,
							currentMD5, currentModTime.Format(time.RFC3339), relativePath)
						dbLock.Unlock()
						resultQueue <- fmt.Sprintf("文件已修改，MD5已更新: %s", filePath)
					}
				}
				bar.Add(1)
			}
		}()
	}

	wg.Wait()
	close(resultQueue)

	for message := range resultQueue {
		fmt.Println(message)
	}

	fmt.Println("\n**MD5计算和验证任务完成**\n")
}

func selectFolder() string {
	var folderSelected string
	fmt.Print("请输入要检查的文件夹路径: ")
	fmt.Scanln(&folderSelected)
	return folderSelected
}

func main() {
	d := selectFolder()
	verifyMD5InFolder(d, 10)
	fmt.Println("按 Enter 键退出...")
	fmt.Scanln()
}
