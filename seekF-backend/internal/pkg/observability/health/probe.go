package health

import (
	"context"
	"sync"
	"time"
)

// Result 表示单项依赖的即时探测结果，不依赖管理端响应类型。
type Result struct {
	Name   string
	Status string
	Detail string
}

// healthProbe 描述单项依赖的只读探测及不暴露内部配置的失败提示。
type healthProbe struct {
	name        string
	unavailable string
	check       func(context.Context) (string, error)
}

// collectDependencyHealth 并行探测依赖，各自限时并按定义顺序收集结果。
func collectDependencyHealth(ctx context.Context, probes []healthProbe) []Result {
	results := make([]Result, len(probes))
	var group sync.WaitGroup
	for index, probe := range probes {
		group.Go(func() {
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			health := Result{Name: probe.name, Status: "down", Detail: probe.unavailable}
			if detail, err := probe.check(probeCtx); err == nil {
				health.Status = "up"
				health.Detail = detail
			}
			// 每个协程仅写入自己的位置，返回顺序不受探测耗时影响。
			results[index] = health
		})
	}
	group.Wait()
	return results
}
