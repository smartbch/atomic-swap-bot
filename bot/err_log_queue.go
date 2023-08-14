package bot

import (
	"sync"
	"time"

	"github.com/zyedidia/generic/queue"
)

type ErrLog struct {
	TS  int64  `json:"ts,omitempty"`
	Lv  string `json:"lv,omitempty"`
	Msg string `json:"msg,omitempty"`
}

type ErrLogQueue struct {
	errLogLimit int
	errLogMutex sync.Mutex
	errLogQueue *queue.Queue[ErrLog]
}

func newErrLogQueue(limit int) *ErrLogQueue {
	return &ErrLogQueue{
		errLogLimit: limit,
		errLogMutex: sync.Mutex{},
		errLogQueue: queue.New[ErrLog](),
	}
}

func (q *ErrLogQueue) recordErrLog(level, errMsg string) {
	q.errLogMutex.Lock()
	defer q.errLogMutex.Unlock()

	if q.errLogQueue.Len() >= q.errLogLimit {
		for i := 0; i < 500; i++ {
			q.errLogQueue.TryDequeue()
		}
	}

	q.errLogQueue.Enqueue(ErrLog{
		TS:  time.Now().Unix(),
		Lv:  level,
		Msg: errMsg,
	})
}

func (q *ErrLogQueue) removeErrLogs(n int) []ErrLog {
	q.errLogMutex.Lock()
	defer q.errLogMutex.Unlock()

	removed := make([]ErrLog, 0, n)
	for i := 0; i < n; i++ {
		if errMsg, ok := q.errLogQueue.TryDequeue(); ok {
			removed = append(removed, errMsg)
		}
	}
	return removed
}
