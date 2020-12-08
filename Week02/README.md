学习笔记

 `G20200607010418`
 
 # error
 
## 核心：
 
 - 减少 `if err != nil` 的出现
 - 让 error 很好地记录信息
 
 ## 标准库
 
 命名规范：对外暴露的预定义 error：包含包的名字 （Sentinel Error，哨兵，需要立刻处理，不建议成为公共 API 的部分）
 
 ```go
 // bufio/bufio.go	
 	ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte")
 	ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune")
 	ErrBufferFull        = errors.New("bufio: buffer full")
 	ErrNegativeCount     = errors.New("bufio: negative count")
 ```
 
 标准库中定义的 error
 
 ```go
 package errors
 
 // New returns an error that formats as the given text.
 // Each call to New returns a distinct error value even if the text is identical.
 func New(text string) error {
 	return &errorString{text}
 }
 
 // errorString is a trivial implementation of error.
 type errorString struct {
 	s string
 }
 
 func (e *errorString) Error() string {
 	return e.s
 }
 ```
 
 `New` 返回指针，这样可以比较不同 error 的地址，而不是值
 
 ## panic & recover
 
* panic 不等同于其它语言的 exception。
 
* exception 的缺陷：如果全局对象被改了一部分值后抛出 exception ，使用者 catch 后不知道全局对象的状态
 
 ### 框架处理 panic 样例
 
* 网络服务框架，使用 middleware 拦截 panic
 
 ```go
 // https://github.com/go-kratos/kratos/blob/master/pkg/net/http/blademaster/recovery.go
 func Recovery() HandlerFunc {
 	return func(c *Context) {
 		defer func() {
 			var rawReq []byte
 			if err := recover(); err != nil {
 				const size = 64 << 10
 				buf := make([]byte, size)
 				buf = buf[:runtime.Stack(buf, false)]
 				if c.Request != nil {
 					rawReq, _ = httputil.DumpRequest(c.Request, false)
 				}
 				pl := fmt.Sprintf("http call panic: %s\n%v\n%s\n", string(rawReq), err, buf)
 				fmt.Fprintf(os.Stderr, pl)
 				log.Error(pl)
 				c.AbortWithStatus(500)
 			}
 		}()
 		c.Next()
 	}
 }
 ```
 
* gin 拦截 panic
 
 ```go
 // https://github.com/gin-gonic/gin/blob/master/gin.go
 // Default returns an Engine instance with the Logger and Recovery middleware already attached.
 func Default() *Engine {
   // ...
 	engine.Use(Logger(), Recovery())
 	return engine
 }
 
 // https://github.com/gin-gonic/gin/blob/master/recovery.go
 func CustomRecoveryWithWriter(out io.Writer, handle RecoveryFunc) HandlerFunc {
 	var logger *log.Logger
 	if out != nil {
 		logger = log.New(out, "\n\n\x1b[31m", log.LstdFlags)
 	}
 	return func(c *Context) {
 		defer func() {
       if err := recover(); err != nil {
         // ....
         stack(3)
         // ....
       }
     }
   }
 }
 
 // stack returns a nicely formatted stack frame, skipping skip frames.
 func stack(skip int) []byte {
 	buf := new(bytes.Buffer) // the returned data
 	// As we loop, we open files and read them. These variables record the currently
 	// loaded file.
 	var lines [][]byte
 	var lastFile string
 	for i := skip; ; i++ { // Skip the expected number of frames
 		pc, file, line, ok := runtime.Caller(i)
 		if !ok {
 			break
 		}
 		// Print this much at least.  If we can't find the source, it won't show.
 		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
 		if file != lastFile {
 			data, err := ioutil.ReadFile(file)
 			if err != nil {
 				continue
 			}
 			lines = bytes.Split(data, []byte{'\n'})
 			lastFile = file
 		}
 		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
 	}
 	return buf.Bytes()
 }
 ```
 
 ### 处理协程的panic
 
* recover 只能处理同一个 goroutine 中的 panic （野生 goroutine）
 
 ```go
 func cannotCatch() {
     defer func() {
         if err := recover(); err != nil {
             fmt.Println("cathc error", err)
         }
     }()
     go func() {
         panic("hhhhh")
     }()
 }
 ```
 
 * 需要自己封装 goroutine 的调用
 
 ```go
 func catched() {
     Go := func(fn func()) {
         go func() {
             defer func() {
                 if err := recover(); err != nil {
                     fmt.Println("cathc error", err)
                 }
             }()
             fn()
         }()
     }
     Go(func() {
         panic("hhhhhh")
     })
 }
 ```
 

**处理request:**
   * ❎  一次请求一个 goroutine
   * ✅ 将请求传到一个 channel 中，由专门的 goroutine 处理
 
 ### 何时不处理 panic
 
 - 强依赖启动失败：
   - 依赖的数据库启动失败？**看场景** ，如果是**读多写少** ，如果可以连接上 cache ，则可以启动
 - 服务配置文件写错
 
 ---
 
 ## 如何暴露 error
 
 **Opaque Errors**
 
 ```go
 type timeout interface {
   Timeout() bool
 }
 
 func IsTimeout(err error) bool {
   to, ok := err.(timeout)
   return ok && to.Timeout()
 }
 ```
 
 对外暴露行为（方法），而不是类型；内部断言
 
 ---
 
 ## 如何设计函数返回 error 的方式
 
 如果一段逻辑需要调用多次相同的函数，则可以考虑对函数进行封装，保存其返回 error 的状态。这样做科研有效地减少 `if err != nil` 的出现，简化代码
 
 ```go
 type errWriter struct {
 	io.Writer
 	err error
 }
 
 func (ew *errWriter) Write(buf []byte) (int, error) {
 	if ew.err != nil {
 		return 0, ew.err
 	}
 	var n int
 	n, ew.err = ew.Writer.Write(buf)
 	return n, ew.err
 }
 
 func (ew *errWriter) Err() error {
 	return ew.err
 }
 
 func main() {
 	writer := bytes.NewBuffer([]byte{})
 	ew := &errWriter{Writer: writer}
 
 	fmt.Fprintf(ew, "aaaa")
 	fmt.Fprintf(ew, "bbbb")
 	fmt.Fprintf(ew, "cccc")
 	if err := ew.Err(); err != nil {
 		panic(err)
 	}
 	fmt.Println(writer.String())
 }
 ```
 
 这种统一处理 error 的方式在标准库中也使用到，比如说 `bufio.Scanner`
 
 ```go
 func countLines(r io.Reader) (lines int, err error) {
   sc := bufio.Scanner(r)
   for sc.Scan() {
     lines++
   }
   return lines, sc.Err()
 }
 ```
 
 
 
 ## Wrap Errors
 
 > You should only handle errors once. Handling an error means inspecting the error value and make a simple decision.
 
 要么直接将错误想上抛出，也就是直接 return error；要么做降级处理，自己处理这个 error 并打日志，继续接下来的逻辑
 
 第三方的库 [github.com/pkg/errors](https://github.com/pkg/errors) 支持打印堆栈
 
 ```go
package main
import (
	"fmt"
	"github.com/pkg/errors"
)

func Func1() error {
	return errors.New("Func1 calling error")
}

func Func2 () error {
	return Func1()
}

func main() {
	fmt.Printf("%+v", Func2())
}
 ```
 
 输出为：
 
 ```txt
Func1 calling error
main.Func1
        /Users/tony/battleField/Go-000/Week01/example.go:8
main.Func2
        /Users/tony/battleField/Go-000/Week01/example.go:12
main.main
        /Users/tony/battleField/Go-000/Week01/example.go:16
runtime.main
        /usr/local/go/src/runtime/proc.go:204
runtime.goexit
        /usr/local/go/src/runtime/asm_amd64.s:1374
 ```
 
 简单的实践为：
 
 如果函数接收的 `error` 为标准库普通的 error ，则建议使用 `Wrap` 或者 `WithStack` 方法从当前位置保存堆栈
 
 作为基础库，暴露给其他调用者的 error 应该是最简单的 error，不应该带上堆栈
 
 如果对 `github.com/pkg/errors` 的 error 进行多次  `Wrap` 或者 `WithStack` ，则最终会打印重复的堆栈，例如：
 
 ```go
package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func Func1() error {
	return errors.New("Func1 calling error")
}

func Func2 () error {
	return errors.Wrap(Func1(), "wrap err in Func2")
}

func main() {
	fmt.Printf("%+v", Func2())
}
 ```
 
 输出为：
 
 ```txt
Func1 calling error
main.Func1
        /Users/tony/battleField/Go-000/Week01/example.go:9
main.Func2
        /Users/tony/battleField/Go-000/Week01/example.go:13
main.main
        /Users/tony/battleField/Go-000/Week01/example.go:17
runtime.main
        /usr/local/go/src/runtime/proc.go:204
runtime.goexit
        /usr/local/go/src/runtime/asm_amd64.s:1374
wrap err in Func2
main.Func2
        /Users/tony/battleField/Go-000/Week01/example.go:13
main.main
        /Users/tony/battleField/Go-000/Week01/example.go:17
runtime.main
        /usr/local/go/src/runtime/proc.go:204
runtime.goexit
        /usr/local/go/src/runtime/asm_amd64.s:1374
 ```
 
 可以使用接口 `Cause`、 `As` 、 `UnWrap` 与 `Is` 还原最初的错误，或者 `%w`，这与 Go 1.13.0 之后给出的接口是兼容的
 
 ---
 
# Week02 作业题目：

1. 我们在数据库操作的时候，比如 dao 层中当遇到一个 sql.ErrNoRows 的时候，是否应该 Wrap 这个 error，抛给上层。为什么，应该怎么做请写出代码？

以上作业，要求提交到 Github 上面，Week02 作业提交地址：
https://github.com/Go-000/Go-000/issues/8

请务必按照示例格式进行提交，不要复制其他同学的格式，以免格式错误无法抓取作业。
作业提交截止时间 12 月 2 日（下周三）23:59 前。

Github 使用教程： https://u.geekbang.org/lesson/51?article=294701

 [点击查看作业](./main.go)
 
 在 dao 层 `Wrap` 一下，在最终的 business 层打印出错的堆栈信息，使用 `Cause` 提取出原始错误，根据其返回给调用者相应的提示信息