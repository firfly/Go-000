package errgroup

import (
	"context"
	"fmt"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// Group 对 sync/errgroup 的简单封装，封装 context，新增方法 StopAll 以及在 Go 中对 panic 做 recover
// 参考 https://github.com/go-kratos/kratos/blob/master/pkg/sync/errgroup/errgroup.go
type Group struct {
	ctx      context.Context
	cancel   func()
	rawGroup *errgroup.Group
}

// WithContext 新建一个 Group
func WithContext(ctx context.Context) *Group {
	c, f := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(c)
	return &Group{ctx: ctx, cancel: f, rawGroup: g}
}

// Go 启动一个任务
func (g *Group) Go(fn func(context context.Context) error) {
	g.rawGroup.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 64<<10)
				buf = buf[:runtime.Stack(buf, false)]
				err = fmt.Errorf("errgroup: panic recovered: %s\n%s", r, buf)
			}
		}()
		return fn(g.ctx)
	})
}

// Wait 等待所有任务的完成
func (g *Group) Wait() error {
	return g.rawGroup.Wait()
}

// StopAll 立刻结束所有的任务
func (g *Group) StopAll() {
	g.cancel()
}