package main

import (
	"bufio"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"goToolkit/Bar"
	"goToolkit/jmath"
	"goToolkit/jpath"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const dbName = "file_integrity_check_by_go.db"
const logName = "file_integrity_check_by_go.log"

var excludeFileNames = []string{dbName, logName}
var dbLock sync.Mutex
var optionMap = map[string]string{
	"1": "全量扫描验证",
	"2": "增量扫描验证",
	"3": "随机抽取验证（1/20）",
	"4": "随机抽取验证（1/10）",
	"5": "随机抽取验证（1/2）",
}

type opRecord struct {
	message      string
	relativePath string
	// 0 其他错误
	// 1 与数据库md5相同
	// 2 文件存在修改，更新md5
	// 3 文件损坏，md5错误
	// 4 新增文件，添加md5
	status int
}

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

func pushFilesToProcess(files []string, taskQueue chan<- string) {
	defer close(taskQueue)
	for _, filePath := range files {
		taskQueue <- filePath
	}
}

func verifyMD5InFolder(mainDirectory string, threadCount int, mode string) {
	taskQueue := make(chan string, 10)
	resultQueue := make(chan opRecord, 10)
	updateCount := 0
	brokenCount := 0
	addCount := 0
	totalCount := 0
	var brokenFiles []string

	filters := []func(db *sql.DB, mainDirectory, fileName, filePath string) int{filterExcludeFileAndSfv}
	if mode == "1" {

	} else if mode == "2" {
		filters = append(filters, filterExistFile)
	} else if mode == "3" {
		filters = append(filters, genRandomFileFilter(16))
	} else if mode == "4" {
		filters = append(filters, genRandomFileFilter(8))
	} else if mode == "5" {
		filters = append(filters, genRandomFileFilter(4))
	} else {
		fmt.Println("工作模式无效")
		return
	}

	start := time.Now()
	defer func() {
		close(resultQueue)
		end := time.Now()
		seconds := end.Sub(start).Seconds()
		fmt.Println("\n****MD5计算和验证任务完成，总耗时为", jmath.RoundWithPrecision(seconds, 2), "秒****")
		// 定义日志文件名
		logFile := path.Join(mainDirectory, logName)
		// 打开文件，如果文件不存在则创建，追加内容
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close() // 确保在函数结束时关闭文件
		// 创建一个新的日志记录器
		logger := log.New(file, "INFO: ", log.Ldate|log.Ltime)
		// 向日志文件中写入内容
		logger.Printf("工作模式: %s, 新增文件数量: %d, 损坏文件数量: %d, 更新文件数量: %d, 总扫描文件数: %d \n", optionMap[mode], addCount, brokenCount, updateCount, totalCount)
		for ss := range brokenFiles {
			logger.Printf("-- %s \n", ss)
		}
	}()

	fmt.Println("初始化数据库")
	dbPath := filepath.Join(mainDirectory, dbName)
	if err := initializeDB(dbPath); err != nil {
		fmt.Println("数据库初始化失败:", err)
		return
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Println("数据库连接失败:", err)
		return
	}
	defer db.Close() // 确保在函数结束时关闭连接

	count, files := countFilesInDirectory(db, mainDirectory, filters...)
	b := Bar.NewBar(0, count)
	go pushFilesToProcess(files, taskQueue)

	var wg sync.WaitGroup

	fmt.Println("开始处理任务队列\n")
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go dealSingleFile(&wg, taskQueue, resultQueue, mainDirectory, db)
	}
	go func() {
		for mess := range resultQueue {
			//fmt.Println(message)
			b.Add(1, mess.message)
			totalCount++
			if mess.status == 2 {
				updateCount++
			} else if mess.status == 3 {
				brokenCount++
				brokenFiles = append(brokenFiles, mess.relativePath)
			} else if mess.status == 4 {
				addCount++
			}
		}
	}()
	wg.Wait()
	time.Sleep(1)
}

// 需要过滤掉的返回1，需要处理的返回0
func filterExcludeFileAndSfv(db *sql.DB, mainDirectory string, fileName string, filePath string) int {
	if filepath.Ext(fileName) == ".sfv" {
		return 1
	}
	for _, exclude := range excludeFileNames {
		if fileName == exclude {
			return 1
		}
	}
	return 0
}

func filterExistFile(db *sql.DB, mainDirectory string, fileName string, filePath string) int {

	relpath, _ := filepath.Rel(mainDirectory, filePath)
	var dbMD5 string
	var dbModTimeStr string
	err := db.QueryRow(`SELECT md5, modification_time FROM files_md5 WHERE file_path = ?`, relpath).Scan(&dbMD5, &dbModTimeStr)
	if err == sql.ErrNoRows {
		return 0
	}
	return 1
}

func genRandomFileFilter(max int) func(db *sql.DB, mainDirectory string, fileName string, filePath string) int {
	return func(db *sql.DB, mainDirectory string, fileName string, filePath string) int {
		ra := rand.Intn(max)
		if ra == 0 {
			return 0
		} else {
			return 1
		}
	}
}

func countFilesInDirectory(db *sql.DB, directory string, filters ...func(db *sql.DB, mainDirectory string, fileName string, filePath string) int) (int, []string) {
	count := 0
	files := make([]string, 0)
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		for _, filter := range filters {
			if filter(db, directory, info.Name(), path) == 1 {
				return nil
			}
		}
		count++
		files = append(files, path)
		return nil
	})
	return count, files
}

func dealSingleFile(wg *sync.WaitGroup, taskQueue chan string, resultQueue chan opRecord, mainDirectory string, db *sql.DB) {
	defer wg.Done()
	for filePath := range taskQueue {
		currentMD5, err := calculateMD5(filePath)
		if err != nil {
			resultQueue <- opRecord{
				status:  0,
				message: fmt.Sprintf("文件 %s 计算MD5失败: %v", filePath, err),
			}
			continue
		}
		currentModTime, err := getFileModificationTime(filePath)
		if err != nil {
			resultQueue <- opRecord{
				status:  0,
				message: fmt.Sprintf("文件 %s 获取修改时间失败: %v", filePath, err),
			}
			continue
		}
		relativePath, err := filepath.Rel(mainDirectory, filePath)
		if err != nil {
			resultQueue <- opRecord{
				status:  0,
				message: fmt.Sprintf("计算相对路径失败: %v", err),
			}
			continue
		}

		var dbMD5 string
		var dbModTimeStr string
		err = db.QueryRow(`SELECT md5, modification_time FROM files_md5 WHERE file_path = ?`, relativePath).Scan(&dbMD5, &dbModTimeStr)

		if err == sql.ErrNoRows {
			resultQueue <- opRecord{
				message: fmt.Sprintf("文件新增: %s", relativePath),
				status:  4,
			}
			dbLock.Lock()
			_, _ = db.Exec(`INSERT INTO files_md5 (file_path, md5, modification_time) VALUES (?, ?, ?)`,
				relativePath, currentMD5, currentModTime)
			dbLock.Unlock()
		} else if err != nil {
			resultQueue <- opRecord{
				message: fmt.Sprintf("查询数据库失败: %v", err),
				status:  0,
			}
		} else {
			if currentModTime == dbModTimeStr {
				if currentMD5 == dbMD5 {
					resultQueue <- opRecord{
						message: fmt.Sprintf("文件: %s", relativePath),
						status:  1,
					}
				} else {
					resultQueue <- opRecord{
						message: fmt.Sprintf("文件损坏: %s", relativePath),
						status:  3,
					}
				}
			} else {
				dbLock.Lock()
				_, _ = db.Exec(`UPDATE files_md5 SET md5 = ?, modification_time = ? WHERE file_path = ?`,
					currentMD5, currentModTime, relativePath)
				dbLock.Unlock()
				resultQueue <- opRecord{
					message: fmt.Sprintf("文件修改: %s", relativePath),
					status:  2,
				}
			}
		}
	}
}

func main() {

	var mode string
	var folder string

	args := os.Args
	fmt.Println(args, len(args))
	if len(args) >= 3 {
		mode = args[1]
		folder = args[2]
		err := jpath.CheckFolderLeagl(&folder)
		if err != nil {
			return
		}
		verifyMD5InFolder(folder, 10, mode)
	} else {
		fmt.Println("请输入工作模式:")
		fmt.Println(" 1. 全量扫描验证")
		fmt.Println(" 2. 增量扫描验证")
		fmt.Println(" 3. 随机抽取验证（1/16概率）")
		fmt.Println(" 4. 随机抽取验证（1/8概率）")
		fmt.Println(" 5. 随机抽取验证（1/4概率）")
		_, err := fmt.Scan(&mode)
		if err != nil {
			return
		}
		err = jpath.InputFolderAndCheck(&folder)
		if err != nil {
			return
		}
		verifyMD5InFolder(folder, 10, mode)
		fmt.Println("按 Enter 键退出...")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
	}
}
