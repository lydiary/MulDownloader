package core

import (
	"sync"
)

type CoroutinePool struct {
	coroutineCount int
	waitGroup      sync.WaitGroup
	taskChan       chan interface{}
}

type PoolHandler func(interface{})

func (pool *CoroutinePool) Wait() {
	pool.waitGroup.Wait()
}

func (pool *CoroutinePool) AddTask(data interface{}) {
	pool.taskChan <- data
}

func CreateCoroutinePool(coroutineCount int, handler PoolHandler) *CoroutinePool {
	pool := &CoroutinePool{}
	pool.taskChan = make(chan interface{}, coroutineCount)
	pool.coroutineCount = coroutineCount

	pool.waitGroup.Add(coroutineCount)

	for i := 0; i < coroutineCount; i++ {
		go func() {
			for {
				data, ok := <-pool.taskChan
				if !ok {
					pool.waitGroup.Done()
				}
				handler(data)
				if len(pool.taskChan) <= 0 {
					pool.waitGroup.Done()
				}
			}
		}()
	}
	return pool
}

func DestroyCoroutinePool(pool *CoroutinePool) {
	pool.Wait()
	close(pool.taskChan)
}
