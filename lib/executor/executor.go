package executor

type Executor struct {
	CommandQueues map[string]WorkQueue
}

func (e Executor) CreateQueue(name string) {
	queue := WorkQueue{}
	e.CommandQueues[name] = queue
}

func (e Executor) AddCommandToQueue(name string) {

}
