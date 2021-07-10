package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	Handler func(v ...interface{})
	Params  []interface{}
}

type Pool struct {
	capacity       uint64 //池的容量
	runningWorkers uint64
	status         int64      //任务池的状态
	chTask         chan *Task //任务队列
	PanicHandler   func(interface{})
	sync.Mutex
}

var ErrInvalidPoolCap = errors.New("invalid pool cap")
var ErrPoolAlreadyClosed = errors.New("pool already closed")

const (
	RUNNING = 1
	STOPED  = 0
)

//任务池初始化
func NewPool(capacity uint64) (*Pool, error) {
	if capacity <= 0 {
		return nil, ErrInvalidPoolCap
	}
	return &Pool{
		capacity: capacity,
		status:   RUNNING,
		chTask:   make(chan *Task, capacity),
	}, nil
}

//启动worker
func (p *Pool) run() {
	p.incRunning()

	go func() {
		defer func() {
			p.decRunning()
			if r := recover(); r != nil {
				if p.PanicHandler != nil {
					p.PanicHandler(r)
				} else {
					log.Printf("Worker panic:%s\n", r)
				}
			}
			p.checkWorker()
		}()

		for {
			select {
			case task, ok := <-p.chTask:
				if !ok {
					return
				}
				task.Handler(task.Params...)
			}
		}
	}()
}

func (p *Pool) incRunning() {
	atomic.AddUint64(&p.runningWorkers, 1)
}

func (p *Pool) decRunning() {
	atomic.AddUint64(&p.runningWorkers, ^uint64(0))
}

func (p *Pool) GetRunningWorkers() uint64 {
	return atomic.LoadUint64(&p.runningWorkers)
}

func (p *Pool) checkWorker() {
	p.Lock()
	defer p.Unlock()

	if p.runningWorkers == 0 && len(p.chTask) > 0 {
		p.run()
	}
}

func (p *Pool) GetCap() uint64 {
	return p.capacity
}

func (p *Pool) setStatus(status int64) bool {
	p.Lock()
	defer p.Unlock()

	if p.status == status {
		return false
	}

	p.status = status
	return true
}

//将任务放到池中
func (p *Pool) Put(task *Task) error {
	p.Lock()
	defer p.Unlock()

	if p.status == STOPED {
		return ErrPoolAlreadyClosed
	}

	if p.GetRunningWorkers() < p.GetCap() {
		//创建启动一个worker
		p.run()
	}

	if p.status == RUNNING {
		p.chTask <- task
	}
	return nil
}

func (p *Pool) Close() {
	p.setStatus(STOPED)

	for len(p.chTask) > 0 { // 阻塞等待所有任务被 worker 消费
		time.Sleep(1e6) // 防止等待任务清空 cpu 负载突然变大, 这里小睡一下
	}

	close(p.chTask)
}

func main() {
	pool,err :=NewPool(10)
	if err!= nil {
		panic(err)
	}

	for i:=0;i<20;i++ {
		pool.Put(&Task{
			Handler: func(v ...interface{}){
				fmt.Println(v)
			},
			Params: []interface{}{i},
		})
	}
	time.Sleep(1e9)
}
