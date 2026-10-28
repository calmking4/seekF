package workerpool

import (
	"context"
	"time"
)

// NewQueue 创建有界任务队列与固定工作协程；timeout为零时不限制任务耗时，关闭队列后排空任务并退出。
func NewQueue[T any](workers, capacity int, timeout time.Duration, handle func(context.Context, T)) chan T {
	if workers <= 0 || capacity < 0 || timeout < 0 || handle == nil {
		panic("协程池参数无效")
	}
	tasks := make(chan T, capacity)
	for i := 0; i < workers; i++ {
		go func() {
			for task := range tasks {
				ctx := context.Background()
				cancel := func() {}
				if timeout > 0 {
					ctx, cancel = context.WithTimeout(ctx, timeout)
				}
				handle(ctx, task)
				cancel()
			}
		}()
	}
	return tasks
}
