package ai

import (
	"context"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/workerpool"
	"seekF-backend/internal/pkg/zlog"
)

// aiCommentReader 只暴露批次调度所需的读取和同步提交能力。
type aiCommentReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
}

// waitAICommentRetry 等待重试或在服务取消后停止等待。
func waitAICommentRetry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// aiCommentJob 保存任务与完成信号，整批完成前不提交位点。
type aiCommentJob struct {
	message kafka.Message
	done    chan struct{}
}

// consumeAICommentBatches 通过固定协程处理有界批次，避免乱序完成时越过未完成的消费位点。
func consumeAICommentBatches(ctx context.Context, reader aiCommentReader, process func(kafka.Message)) {
	jobs := workerpool.NewQueue(constants.AICommentWorkers, constants.AICommentWorkers, 0, func(_ context.Context, job aiCommentJob) {
		process(job.message)
		close(job.done)
	})
	defer close(jobs)
	for ctx.Err() == nil {
		var batch []aiCommentJob
		for len(batch) < constants.AICommentWorkers {
			fetchCtx := ctx
			cancel := func() {}
			if len(batch) > 0 {
				fetchCtx, cancel = context.WithTimeout(ctx, constants.AICommentBatchWait)
			}
			message, err := reader.FetchMessage(fetchCtx)
			cancel()
			if err != nil {
				if !errors.Is(err, context.DeadlineExceeded) {
					zlog.Error("从Kafka读取AI评论任务失败: " + err.Error())
					if !waitAICommentRetry(ctx) {
						return
					}
				}
				break
			}
			job := aiCommentJob{message: message, done: make(chan struct{})}
			batch = append(batch, job)
			jobs <- job
		}
		var messages []kafka.Message
		for _, job := range batch {
			<-job.done
			messages = append(messages, job.message)
		}
		if len(messages) == 0 {
			continue
		}
		// 只在整个批次完成后提交，失败时只重试提交，不重新生成回复。
		for ctx.Err() == nil {
			if err := reader.CommitMessages(ctx, messages...); err == nil {
				break
			} else {
				zlog.Error("提交AI评论消费位点失败: " + err.Error())
				if !waitAICommentRetry(ctx) {
					return
				}
			}
		}
	}
}
