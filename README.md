# go协程池实现

实现细节：控制系统中协程的数量

```
//任务
type Task struct {
	Handler func(v ...interface{})
	Params  []interface{}
}

//协程池
type Pool struct {
	capacity       uint64 //池的容量
	runningWorkers uint64 //运行中的协程数
	status         int64      //任务池的状态
	chTask         chan *Task //任务队列
	PanicHandler   func(interface{})
	sync.Mutex
}
```
