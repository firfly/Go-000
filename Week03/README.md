学习笔记

👉Week03 作业题目：
1.基于 errgroup 实现一个 http server 的启动和关闭 ，以及 linux signal 信号的注册和处理，要保证能够 一个退出，全部注销退出。

[点击查看作业](./main.go)

# Concurrency

- 原子
- 可见性

## goroutine

- 线程、进程，内核态
- goroutine，用户态，一个 P 和 M 可能对应多个 G

并发不是并行

做代码的超时控制（防止 goroutine 泄漏）

```go
func main() {
  ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
  defer cancel()
  
  ch := make(chan result)
  go func() {
    record, err := search(ctx)
    ch <- result{record, err}
  }()
  
  select {
  case <- ctx.Done():
    // timeout
  case res := <-ch:
    // get result
  }
}
```

Keep yourself busy or do the work yourself 不建议将阻塞的函数新开一个 goroutine 却在主函数中用 `select` 进行阻塞。而是将阻塞函数放在主函数中执行

```go
// ❎
func main() {
  go server.SrvHTTP()
  select{}
}
```

```go
// ✅
func main() {
  server.SrvHTTP()
}
```

Leave concurrency to the caller **某些接口由用户决定是否要并发执行**

```go
func ListDir(dir string) ([]string, error)
func ListDir(dir string) chan string
func ListDir(dir string, func(string)error) error
```

```go
func main() {
  ListDir(".", func(name string) error {
    process(name)
    if 处理结束了，希望中途退出 {
      return errors.New("quit!!")
    }
    return nil
  })
}
```

Never start a goroutine without knowning when it will stop。需要对自己开出的 goroutine 负责，也就是**知道其会在何时退出**，**可以控制其何时退出**。例如如何感知到位于 goroutine 中的 http server 退出，如何防止其退出。不建议粗暴地使用 `log.Fatal` 来退出，因为`log.Fatal` 直接调用 `os.Exit` ，会导致 `defer` 函数无法执行；一般只在 main 或 init 中使用

```go
func server(stop <-chan struct{}) error {
  go func() {
    <- stop
    s.Shutdown()
  }
  return s.Serve()
}

func main() {
  done := make(chan error, 2)
  stop := make(chan struct{})
  go func() {
    done <- server(stop)
  }()
  go func() {
    done <- server(stop)
  }()
  var stopped bool
  for i := 0; i < cap(done); i++ {
    if err := <- done; err != nil {
      log.Println(err)
    }
    if !stopped {
      stopped = true
      close(stop)
    }
  }
}
```

Demo: 可控日志

不建议 `go t.Event` ，而是采用 channel 的方式消费消息，可以控制 goroutine 数量以及状态

```go
func main() {
	t := NewTracker()
	go t.Run()
	_ = t.Event(context.Background(), "test1")
	_ = t.Event(context.Background(), "test2")
	_ = t.Event(context.Background(), "test3")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	defer cancel()
	t.Shutdown(ctx)
}

func NewTracker() *Tracker {
	return &Tracker{ch: make(chan string, 2), stop: make(chan struct{})}
}

// Tracker 可控的日志记录
type Tracker struct {
	ch   chan string
	stop chan struct{}
}

func (t *Tracker) Event(ctx context.Context, data string) error {
	select {
	// case <-t.stop:
	// 	return nil
	case t.ch <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *Tracker) Run() {
	for data := range t.ch {
		time.Sleep(time.Second)
		fmt.Println(data)
	}
	t.stop <- struct{}{}
}

func (t *Tracker) Shutdown(ctx context.Context) {
	close(t.ch)
	select {
	case <-t.stop:
	case <-ctx.Done():
	}
}
```

---

## 内存模型

https://golang.org/ref/mem

happen-before

内存重排：优化

多线程环境下，CPU的缓存来不及写到内存中，无法读到数据

```txt
线程1    |     线程2
 A=1            B=1
print(B)      print(A)

-->

线程1    |     线程2
var A         var B
print(B)      print(A)
 A=1           B=1
```

barrier/fence：要求所有对内存的操作都必须要“扩散”到内存之后才能继续执行其他对内存的操作 （将缓存及时同步到内存）

single machine word：64位机器 --> 8 byte --> 一次性读取

指针不一定是 single machine word ，例如 interface 中包含了两个指针，指向 data 和 type。（内存布局）

原子性：一次性操作，32位CPU加载 8byte指针需要加载两次，不是原子性，而64位机器一次性就可以了

可见性：由于有多级 cache 的存在，所以即使能保证原子性也不一定能保证可见性

## sync

### sync.Atomic

Copy-On-Write: 写操作时候复制全量老数据到一个新的对象中，携带上本次新写的数据，之后利用原子替换(atomic.Value)，更新调用者的变量。(Redis COW BGSave)

```go
func main() {
  var config atomic.Value
  config.Store(loadConfig())
  go func() {
    for {
      time.Sleep(time.Minute)
      config.Store(loadConfig()) // 写
    }
  }()
  for i := 0; i < 10; i++ {
    go func() {
      for r := range requests() {
        c := config.Load() // 读
        // do something
      }
    }()
  }
}
```

### sync.Mutex

几种锁的类型

- **Barging**：提高吞吐量，唤醒第一个等待者，把锁给第一个等待者或者给第一个请求锁的人。可能会造成饥饿
- **Handsoff**：锁释放时候，锁会一直持有直到第一个等待者准备好获取锁。吞吐量降低
- **Spinning**: for 死循环，尝试获取锁，防止休眠。降低上下文切换开销（parking and unparking goroutine）。等待队列为空或者应用程序重度使用锁时效果不错。
  - **Nginx pause 优化**

Golang 1.9之前：Barging + Spinning。当试图获取已经被持有的锁时，如果本地队列为空并且 P 的数量大于1，goroutine 将自旋几次。

Golang 1.9：添加饥饿模式。所有等待锁超过一毫秒的 goroutine(也称为有界等待)将被诊断为饥饿，停用 Spinning，unlock 方法会 handsoff 把锁直接扔给第一个等待者

### errgroup

[https://pkg.go.dev/golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)

内置 `sync.WaitGroup`来控制并发进程

返回第一个出的错

[https://github.com/go-kratos/kratos/tree/master/pkg/sync/errgroup](https://github.com/go-kratos/kratos/tree/master/pkg/sync/errgroup)

优化：

初始化时的 `context` 不能在其他地方使用，防止被意外 cancel

需要控制好开出的 goroutine 个数

需要控制好 errgroup 开出的野生 goroutine

### sync.Pool

保存和复用临时对象，以减少内存分配，降低 GC 压力

## chan

无缓冲 channel 的本质是保证同步

## Context

将 `context` 作为函数第一个参数传递

起始使用 `context.Background()`

不要修改 `context.Value()` 的值（应该是只读的） ，如果想修改，则调用 `context.WithValue()` 再包一层 （COW）

`context.Value()` 中存放的值应该与业务无关 （Context.Value should inform, not control）

跨 goroutine 超时处理

```go
// 超时时间根据当前服务的配置以及上下文生成
func shrinkDeadline(ctx context.Context, timeout time.Duration) time.Time {
	timeoutTime := time.Now().Add(timeout)
	if deadline, ok := ctx.Deadline(); ok && timeoutTime.After(deadline) {
		return deadline
	}
	return timeoutTime
}
```



## 阅读

https://golang.org/ref/mem

https://www.jianshu.com/p/5e44168f47a3

https://mp.weixin.qq.com/s/vnm9yztpfYA4w-IM6XqyIA

https://wudaijun.com/2019/04/cpu-cache-and-memory-model/

mysql double write buffer 

1.13+ 的 sync.Pool

context 代码

https://talks.golang.org/2014/gotham-context.slide#1

https://blog.golang.org/concurrency-timeouts
https://blog.golang.org/pipelines
https://talks.golang.org/2013/advconc.slide#1
https://github.com/go-kratos/kratos/tree/master/pkg/sync
