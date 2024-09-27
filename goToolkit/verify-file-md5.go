package main

import (
	"bufio"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"goToolkit/jmath"
	jpath "goToolkit/jpath"
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
	var h = md5.New()
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

func getFileModificationTime(filePath string) (string, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	return stat.ModTime().Format(time.DateTime), nil
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
	defer close(taskQueue)
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
	taskQueue := make(chan string, 10)
	resultQueue := make(chan string, 10)

	start := time.Now()
	defer func() {
		end := time.Now()
		seconds := end.Sub(start).Seconds()
		fmt.Println("总耗时为", jmath.RoundWithPrecision(seconds, 2), "秒")
	}()

	fmt.Println("初始化数据库")
	dbPath := filepath.Join(mainDirectory, dbName)
	if err := initializeDB(dbPath); err != nil {
		fmt.Println("数据库初始化失败:", err)
		return
	}

	go pushFilesToProcess(mainDirectory, taskQueue)

	var wg sync.WaitGroup
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Println("数据库连接失败:", err)
		return
	}
	defer db.Close() // 确保在函数结束时关闭连接

	fmt.Println("开始处理任务队列\n")
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go dealSingleFile(&wg, taskQueue, resultQueue, mainDirectory, db)
	}
	go func() {
		for message := range resultQueue {
			fmt.Println(message)
		}
	}()
	wg.Wait()
	// 确保以下输出在过程输出之后
	time.Sleep(1)
	close(resultQueue)
	fmt.Println("\n**MD5计算和验证任务完成**")
}

func dealSingleFile(wg *sync.WaitGroup, taskQueue, resultQueue chan string, mainDirectory string, db *sql.DB) {
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
		relativePath, err := filepath.Rel(mainDirectory, filePath)
		if err != nil {
			resultQueue <- fmt.Sprintf("计算相对路径失败: %v", err)
			continue
		}

		var dbMD5 string
		var dbModTimeStr string
		err = db.QueryRow(`SELECT md5, modification_time FROM files_md5 WHERE file_path = ?`, relativePath).Scan(&dbMD5, &dbModTimeStr)

		if err == sql.ErrNoRows {
			resultQueue <- fmt.Sprintf("未找到对应的MD5记录，已添加: %s", filePath)
			dbLock.Lock()
			_, _ = db.Exec(`INSERT INTO files_md5 (file_path, md5, modification_time) VALUES (?, ?, ?)`,
				relativePath, currentMD5, currentModTime)
			dbLock.Unlock()
		} else if err != nil {
			resultQueue <- fmt.Sprintf("查询数据库失败: %v", err)
		} else {
			if currentModTime == dbModTimeStr {
				if currentMD5 == dbMD5 {
					resultQueue <- fmt.Sprintf("MD5与数据库一致: %s", filePath)
				} else {
					resultQueue <- fmt.Sprintf("文件损坏: %s", filePath)
				}
			} else {
				dbLock.Lock()
				_, _ = db.Exec(`UPDATE files_md5 SET md5 = ?, modification_time = ? WHERE file_path = ?`,
					currentMD5, currentModTime, relativePath)
				dbLock.Unlock()
				resultQueue <- fmt.Sprintf("文件已修改，MD5已更新: %s", filePath)
			}
		}
	}
}

func main() {
	var folder string
	err := jpath.InputFolderAndCheck(&folder)
	if err != nil {
		return
	}
	verifyMD5InFolder(folder, 10)
	fmt.Println("按 Enter 键退出...")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input, _ = reader.ReadString('\n')
	fmt.Println(input)
}
