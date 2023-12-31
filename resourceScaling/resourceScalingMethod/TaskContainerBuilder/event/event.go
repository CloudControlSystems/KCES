package event

import (
	"TaskContainerBuilder/messageProto/TaskContainerBuilder"
)

//Define workflow structure
type WorkflowTask struct {
	//workflowID
	WorkflowId string
	//taskNum
	TaskNum uint32
	//taskName
	TaskName string
	//task image
	Image string
	// millicore(1Core=1000millicore)
	Cpu uint64
	// MiB
	Mem uint64
	//task execution order
	TaskOrder uint32
	//Input the env into Pod
	Env map[string]string
	// Input Vector
	InputVector []string
	// Out Vector
	OutputVector []string
	// task pod parameter
	Args []string
	//pod label
	Labels map[string]string
	//duration (second)
	Duration uint64
	//the minimum cpu in task container used by Stress tool
	MinCpu uint64
	//the minimum memory in task container used by Stress tool
	MinMem uint64
}
type workflowTask WorkflowTask
type  WorkflowTaskMap map[uint32] WorkflowTask
var workflowTaskMap = make(map[uint32] WorkflowTask)

var eventByName = make(map[string][]func(interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error))

// Register the events
//func RegisterEvent(name string, callback eventBuilder) {
func RegisterEvent(name string, callback func(interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error)) {
	// Find event list by name
	list := eventByName[name]

	// add function in list slice
	list = append(list, callback)

	// store
	eventByName[name] = list
	//log.Println(eventByName)
}


// invoke event
func CallEvent(name string, param interface{}) {

	// Find the event list by name
	list := eventByName[name]
	//log.Println(list)
	// Traverse all the event callback
	for _, callback := range list {

		// Input parameter
		callback(param)

	}
}

