package Bar

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

type Bar struct {
	mu      sync.Mutex
	graph   string    // 显示符号
	rate    string    // 进度条
	percent int       // 百分比
	current int       // 当前进度位置
	total   int       // 总进度
	start   time.Time // 开始时间
}

func NewBar(current, total int) *Bar {
	bar := new(Bar)
	bar.current = current
	bar.total = total
	bar.start = time.Now()
	if bar.graph == "" {
		bar.graph = "█"
	}
	bar.percent = bar.getPercent()
	for i := 0; i < bar.percent; i += 2 {
		bar.rate += bar.graph //初始化进度条位置
	}
	return bar
}

func NewBarWithGraph(start, total int, graph string) *Bar {
	bar := NewBar(start, total)
	bar.graph = graph
	return bar
}

func (bar *Bar) getPercent() int {
	return int((float64(bar.current) / float64(bar.total)) * 100)
}

func (bar *Bar) getTime() (s string) {
	u := time.Now().Sub(bar.start).Seconds()
	h := int(u) / 3600
	m := int(u) % 3600 / 60
	if h > 0 {
		s += strconv.Itoa(h) + "h "
	}
	if h > 0 || m > 0 {
		s += strconv.Itoa(m) + "m "
	}
	s += strconv.Itoa(int(u)%60) + "s"
	return
}

func (bar *Bar) load(msg string) {
	last := bar.percent
	bar.percent = bar.getPercent()
	if bar.percent != last && bar.percent%2 == 0 {
		bar.rate += bar.graph
	}
	fmt.Printf("\r[%-50s]% 3d%%    %2s   %d/%d %s", bar.rate, bar.percent, bar.getTime(), bar.current, bar.total, msg)
	os.Stdout.Sync() // 强制刷新输出缓冲区
}

func (bar *Bar) Reset(current int) {
	bar.mu.Lock()
	defer bar.mu.Unlock()
	bar.current = current
	bar.load("")

}

func (bar *Bar) Add(i int, msg string) {
	bar.mu.Lock()
	defer bar.mu.Unlock()
	bar.current += i
	bar.load(msg)
}

func main() {

	b := NewBar(0, 1000)
	for i := 0; i < 1000; i++ {
		b.Add(1, "")
		time.Sleep(time.Millisecond * 10)
	}

}
