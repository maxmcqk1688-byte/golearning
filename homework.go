package main

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// ==========================================
// 1. 指针 (Pointers)
// ==========================================

// 题目 1: 指针修改单个整数值
func addTen(val *int) {
	if val == nil {
		return
	}
	*val += 10
}

// 题目 2: 修改切片指针指向的切片元素
func doubleSliceElements(slicePtr *[]int) {
	if slicePtr == nil {
		return
	}
	// 解引用获取切片本身，遍历修改其元素
	slice := *slicePtr
	for i := range slice {
		slice[i] *= 2
	}
}

// ==========================================
// 2. Goroutine
// ==========================================

// 题目 1: 启动两个协程交替打印奇偶数
func runOddEvenGoroutines() {
	var wg sync.WaitGroup
	wg.Add(2)

	// 打印奇数
	go func() {
		defer wg.Done()
		for i := 1; i <= 10; i += 2 {
			fmt.Printf("奇数协程: %d\n", i)
			time.Sleep(10 * time.Millisecond) // 稍微让出 CPU 模拟交替效果
		}
	}()

	// 打印偶数
	go func() {
		defer wg.Done()
		for i := 2; i <= 10; i += 2 {
			fmt.Printf("偶数协程: %d\n", i)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()
}

// Task 定义为无参数的函数
type Task func()

// 题目 2: 并发任务调度器
func runTaskScheduler(tasks []Task) {
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		// 显式传入任务 ID，避免闭包捕获循环变量问题
		go func(id int, t Task) {
			defer wg.Done()
			start := time.Now()
			t() // 执行任务
			elapsed := time.Since(start)
			fmt.Printf("任务 [%d] 执行完毕，耗时: %v\n", id, elapsed)
		}(i+1, task)
	}

	wg.Wait()
}

// ==========================================
// 3. 面向对象 (OOP)
// ==========================================

// Shape 接口定义
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Rectangle 矩形
type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

// Circle 圆形
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

// Person 基础结构体
type Person struct {
	Name string
	Age  int
}

// Employee 结构体（组合 Person）
type Employee struct {
	Person     // 嵌入结构体（组合）
	EmployeeID string
}

func (e Employee) PrintInfo() {
	fmt.Printf("员工 ID: %s | 姓名: %s | 年龄: %d\n", e.EmployeeID, e.Name, e.Age)
}

// ==========================================
// 4. Channel
// ==========================================

// 题目 1: 无缓冲通道通信
func runUnbufferedChannel() {
	ch := make(chan int)

	// 生产者协程
	go func() {
		for i := 1; i <= 10; i++ {
			ch <- i
		}
		close(ch) // 发送完毕后关闭通道
	}()

	// 消费者协程（主协程消费）
	for num := range ch {
		fmt.Printf("无缓冲 Channel 接收: %d\n", num)
	}
}

// 题目 2: 缓冲通道通信
func runBufferedChannel() {
	// 创建容量为 10 的缓冲通道
	ch := make(chan int, 10)
	var wg sync.WaitGroup

	wg.Add(2)

	// 生产者协程
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i++ {
			ch <- i
		}
		close(ch)
	}()

	// 消费者协程
	go func() {
		defer wg.Done()
		for num := range ch {
			fmt.Printf("缓冲 Channel 接收: %d ", num)
			if num%10 == 0 {
				fmt.Println() // 每 10 个数换一行
			}
		}
	}()

	wg.Wait()
}

// ==========================================
// 5. 锁机制 (Locking Mechanisms)
// ==========================================

// 题目 1: 使用 sync.Mutex 保护互斥计数器
func runMutexCounter() {
	var count int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				mu.Lock()
				count++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("Mutex 计数器最终结果 (期望 10000): %d\n", count)
}

// 题目 2: 使用 sync/atomic 实现无锁计数器
func runAtomicCounter() {
	var count int64
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				atomic.AddInt64(&count, 1)
			}
		}()
	}

	wg.Wait()
	fmt.Printf("Atomic 无锁计数器最终结果 (期望 10000): %d\n", atomic.LoadInt64(&count))
}

// ==========================================
// 主函数入口 (Main Function)
// ==========================================

func main() {
	fmt.Println("========== 1. 指针 ==========")
	// 指针 题目 1
	n := 5
	addTen(&n)
	fmt.Printf("[1.1] 指针增加10: %d\n", n)

	// 指针 题目 2
	nums := []int{1, 2, 3, 4, 5}
	doubleSliceElements(&nums)
	fmt.Printf("[1.2] 切片元素乘以2: %v\n", nums)

	fmt.Println("\n========== 2. Goroutine ==========")
	// Goroutine 题目 1
	fmt.Println("[2.1] 启动奇偶数协程:")
	runOddEvenGoroutines()

	// Goroutine 题目 2
	fmt.Println("[2.2] 并发任务调度器:")
	tasks := []Task{
		func() { time.Sleep(100 * time.Millisecond) },
		func() { time.Sleep(200 * time.Millisecond) },
		func() { time.Sleep(50 * time.Millisecond) },
	}
	runTaskScheduler(tasks)

	fmt.Println("\n========== 3. 面向对象 ==========")
	// 面向对象 题目 1
	shapes := []Shape{
		Rectangle{Width: 10, Height: 5},
		Circle{Radius: 3},
	}
	for _, shape := range shapes {
		fmt.Printf("[3.1] 形状: %T | 面积: %.2f | 周长: %.2f\n", shape, shape.Area(), shape.Perimeter())
	}

	// 面向对象 题目 2
	emp := Employee{
		Person:     Person{Name: "张三", Age: 28},
		EmployeeID: "EMP-1001",
	}
	fmt.Print("[3.2] ")
	emp.PrintInfo()

	fmt.Println("\n========== 4. Channel ==========")
	// Channel 题目 1
	fmt.Println("[4.1] 无缓冲通道通信:")
	runUnbufferedChannel()

	// Channel 题目 2
	fmt.Println("[4.2] 缓冲通道通信:")
	runBufferedChannel()

	fmt.Println("\n========== 5. 锁机制 ==========")
	// 锁机制 题目 1
	runMutexCounter()

	// 锁机制 题目 2
	runAtomicCounter()
}
