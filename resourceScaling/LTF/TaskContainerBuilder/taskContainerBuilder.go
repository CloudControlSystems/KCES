package main

import (
	"TaskContainerBuilder/event"
	informer "TaskContainerBuilder/informer"
	k8sResource "TaskContainerBuilder/k8sResource"
	"TaskContainerBuilder/messageProto/TaskContainerBuilder"
	"encoding/json"
	_ "flag"
	_ "fmt"
	_ "github.com/fsnotify/fsnotify"
	"github.com/gomodule/redigo/redis"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	_ "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v2 "k8s.io/client-go/listers/core/v1"
	_ "k8s.io/client-go/tools/cache"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)
var clientset *kubernetes.Clientset
type resourceAllocation struct {
	MilliCpu uint64
	Memory uint64
}
type requestResourceConfig struct {
	Timestamp int64
	AliveStatus bool
	ResourceDemand resourceAllocation
}
type NodeAllocateResource struct {
	MilliCpu uint64
	Memory uint64
}
type NodeUsedResource struct {
	MilliCpu uint64
	Memory uint64
}

type NodeUsedResourceMap map[string]NodeUsedResource
type NodeAllocateResourceMap map[string]NodeAllocateResource
//the structure of workflow task
type WorkflowTask struct {
	//workflow ID
	WorkflowId string
	//taskNum
	TaskNum uint32
	//taskName
	TaskName string
	//task Image
	Image string
	//unit millicore(1Core=1000millicore)
	Cpu uint64
	//unit MiB
	Mem uint64
	//task order in workflow
	TaskOrder uint32
	//env parameters being injected into Pod
	Env map[string]string
	// input Vector
	InputVector []string
	// output Vector
	OutputVector []string
	//input parameter
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

//type nodeResidualResource struct {
//	MilliCpu int64
//	Memory int64
//}
var nodeResidualResource k8sResource.NodeResidualResource
//type residualResourceMap map[string]nodeResidualResource
var nodeResidualResourceMap k8sResource.ResidualResourceMap
var nodeResidualMap k8sResource.ResidualResourceMap

var allocateResourceMap k8sResource.NodeAllocateResourceMap
var usedResourceMap k8sResource.NodeUsedResourceMap
//var nodeAllocateResourceMap k8sResource.ResidualResourceMap
//var nodeUsedResourceMap k8sResource.ResidualResourceMap

var taskNsNum int64 =0
var taskPodNum int64 = 0

var podLister v2.PodLister
var nodeLister v2.NodeLister
var namespaceLister v2.NamespaceLister
var taskSema = make(chan int, 1)
var redisSema = make(chan int, 1)


type  WorkflowTaskMap map[uint32] WorkflowTask
var workflowTaskMap = make(map[uint32] WorkflowTask)
var workflowMap = make(map[uint32] map[uint32] WorkflowTask)

var taskReceive = make(chan int, 1)
var thisTaskPodExist bool

var dependencyMap = make(map[uint32]map[string][]string)
var taskCompletedStateMap = make(map[uint32]bool)
var wfTaskCompletedStateMap = make(map[uint32] map[uint32]bool)

var clusterAllocatedCpu uint64
var clusterAllocatedMemory uint64
var clusterUsedCpu uint64
var clusterUsedMemory uint64
var masterIp string
var gatherTime string
var interval uint32
var redisIpPort string

type saveWorkflowTaskData struct {
	StartTime uint64
	Duration uint64
	Deadline uint64
	AliveStatus bool
	MilliCpu uint64
	Memory uint64
	Labels map[string]string
}

var experimentalDataObj *os.File

type Builder func(taskObject WorkflowTask) (*TaskContainerBuilder.InputWorkflowTaskResponse,error)

//resource service structure
type ResourceServiceImpl struct {

}

//workflow input interface module, received requests of workflow task generation from
//workflow injection module via gRPC
func (rs *ResourceServiceImpl) InputWorkflowTask(ctx context.Context,request *TaskContainerBuilder.InputWorkflowTaskRequest)(*TaskContainerBuilder.InputWorkflowTaskResponse,error) {
	var response TaskContainerBuilder.InputWorkflowTaskResponse
	log.Printf("--------------The reception time of this workflow task is: %vms",time.Now().UnixNano()/1e6)

	/*Create map[uint32]bool and identify the execution status of each task, initially as false*/
	taskCompletedStateMap[request.TaskOrder] = false

	//Receiving workflow tasks, packed into workflowMap
	taskReceive <- 1
	workflowTaskMap[request.TaskOrder] = WorkflowTask{
		WorkflowId: request.WorkflowId,
		TaskNum: request.TaskNum,
		TaskName:   request.TaskName,
		Image:      request.Image,
		Cpu:        request.Cpu,
		Mem:        request.Mem,
		TaskOrder:  request.TaskOrder,
		Env:        request.Env,
		InputVector: request.InputVector,
		OutputVector: request.OutputVector,
		Args: request.Args,
		Labels: request.Labels,
		Duration: request.DurationTime,
		MinCpu: request.MinCpu,
		MinMem: request.MinMem,
	}
	<- taskReceive
	log.Println(workflowTaskMap[request.TaskOrder])

    //update the start time and deadline of tasks.
	writeDataInRedis(workflowTaskMap[request.TaskOrder],redisIpPort)

	/*If it is the first task in this workflow, invoke CreateTask function to generate task container.
	If not, invoke CreateTask function to generate task container through Update event of Informer.*/
	if request.TaskOrder == 0 && request.InputVector == nil {
		log.Printf("--------------Starting to run the first task [%d] in current workflow is: %vms",request.TaskOrder,time.Now().UnixNano()/1e6)
        //write experimental data into /home/exp.txt.
		//outData := "First task" + strconv.Itoa(int(request.TaskOrder)) + ": " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + "\n"
		//experimentalDataObj.WriteString(outData)

		res, err := CreateTask(request.TaskName)
		if err != nil {
			panic(err.Error())
		}
		log.Printf("This is the first task: %v, order: %v , workflow: %v, InputVector: %v, OuputVector: %v.\n",request.TaskName,
			request.TaskOrder,request.WorkflowId,request.InputVector,request.OutputVector)
		log.Println("*************************************************")
		response = *res
		return &response, nil
	}
	//package the workflowMap and wfTaskCompletedStateMap
	if  request.OutputVector == nil {
		splitName :=  strings.Split(request.WorkflowId,"-")
		wfId, err := strconv.Atoi(splitName[1])
		if err != nil {
			panic(err.Error())
		}
		workflowMap[uint32(wfId)] = workflowTaskMap
		log.Println(workflowMap)
		wfTaskCompletedStateMap[uint32(wfId)] = taskCompletedStateMap
		log.Println(wfTaskCompletedStateMap)
		workflowTaskMap = make(map[uint32] WorkflowTask)
		taskCompletedStateMap =  make(map[uint32]bool)
	}

	log.Printf("The current taskName: %v: order: %v of workflow: %v.\n",request.TaskName,request.TaskOrder,request.WorkflowId)
	//write experimental data into /home/exp.txt.
	outData := "Receiving " + request.TaskName + " : " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) +"\n"
	experimentalDataObj.WriteString(outData)
	log.Println("--------------------------------------------")
	response.ErrNo = 0
	return &response, nil
}

func writeDataInRedisFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from writeDataInRedisFail()", r)
	}
}
func writeDataInRedis(task WorkflowTask, redisServer string){
	var uploadTaskData saveWorkflowTaskData
	var unMarshalData saveWorkflowTaskData
	defer writeDataInRedisFail()
	// build connection
	log.Printf("redisServer: %v\n",redisServer)
	conn, err := redis.Dial("tcp", redisServer)
	if err != nil {
		log.Println("redis.Dial err=", err)
		return
	}
	//encapsulate data
	//taskID
	taskName := task.TaskName
	log.Printf("task.TaskOrder: %v, task.InputVector: %v\n",task.TaskOrder,task.InputVector)

	//if task.TaskOrder == uint32(0) && task.InputVector == nil {
	if task.TaskOrder == 0 {
		//This is the first task that does not contain the ancestor task.
		start := uint64(time.Now().UnixNano()/1e6)
		uploadTaskData = saveWorkflowTaskData{
			StartTime: start,
			//the task-emulator container implements double 'task.Duration' via the stress tool.
			Duration: task.Duration*1000,
			Deadline: start + task.Duration*1000,
			AliveStatus: false,
			MilliCpu: task.Cpu,
			Memory: task.Mem,
			Labels: task.Labels,
		}
		//write workflow task's data into Redis.
		outData := "WriteIntoRedis:" + task.TaskName +":"+ " StartTime:"  + strconv.Itoa(int(uploadTaskData.StartTime)) + ": Deadline:" + strconv.Itoa(int(uploadTaskData.Deadline)) +"\n"
		experimentalDataObj.WriteString(outData)
		//log.Printf("task0-Data: %v\n",uploadTaskData)

	} else {
		tempTime := uint64(1)
		for _, inputIndex := range task.InputVector {
			//log.Printf("task.InputVector: %v\n",task.InputVector)
			//Find the largest deadline within 'task.InputVector' when have many ancestor tasks.
			taskNa := task.WorkflowId + "-task-" + inputIndex
			// read task data from redis
			r, err := redis.Bytes(conn.Do("get", taskNa))
			if err != nil {
				log.Println("get err=", err)
				return
			}

			err = json.Unmarshal(r,&unMarshalData)
			if err != nil {
				log.Println("json unMarshal is err: ", err)
				return
			}
			if (tempTime < unMarshalData.Deadline)&&(unMarshalData.Deadline != 0){
				tempTime = unMarshalData.Deadline
			}
		}
		uploadTaskData = saveWorkflowTaskData{
			StartTime: tempTime,
			//the task-emulator container implements double 'task.Duration' via the stress tool.
			Duration: task.Duration*1000,
			Deadline: tempTime + task.Duration*1000,
			AliveStatus: false,
			MilliCpu: task.Cpu,
			Memory: task.Mem,
			Labels: task.Labels,
		}
		outData := "WriteIntoRedis:" + task.TaskName +":"+ "StartTime: "  + strconv.Itoa(int(uploadTaskData.StartTime)) + ": Deadline:" + strconv.Itoa(int(uploadTaskData.Deadline)) +"\n"
		experimentalDataObj.WriteString(outData)
	}
	//结构体序列化为字符串
	log.Printf("task-Data: %v\n",uploadTaskData)
	taskData, err := json.Marshal(uploadTaskData)
	// 通过go向redis写入数据 string [key - value]

	_, err = conn.Do("set", taskName, taskData)
	if err != nil {
		log.Println("set err=", err)
		return
	}
	// 关闭连接
	defer conn.Close()
}

func recoverDeleteWorkflowNamespace() {
	if r := recover(); r!= nil {
		log.Println("recovered from DeleteWorkflowNamespace()", r)
	}
}

//Delete workflow namespace
func DeleteWorkflowNamespace(task WorkflowTask) error {
	defer recoverDeleteWorkflowNamespace()
	workflowTaskId := task.WorkflowId
	namespacesClient := clientset.CoreV1().Namespaces()
	deletePolicy := metav1.DeletePropagationForeground
	if err := namespacesClient.Delete(workflowTaskId, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
		return err
	}
	log.Printf("Delete the Namespace: %s.\n", workflowTaskId)
	log.Printf("--------------Delete the Namespace %s in time: %vms.\n", workflowTaskId,time.Now().UnixNano()/1e6)
	//write experimental data into /home/exp.txt.
	outData := "Delete the Namespace " + workflowTaskId + " : " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + "\n"
	experimentalDataObj.WriteString(outData)
	return nil
}

func recoverDeleteCurrentTaskContainer() {
	if r := recover(); r!= nil {
		log.Println("recovered from DeleteCurrentTaskContainer()", r)
	}
}

var deletedPodNum int64 = 0
func DeleteCurrentTaskContainer(param interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error) {
	defer recoverDeleteCurrentTaskContainer()
	var taskPodResponse TaskContainerBuilder.InputWorkflowTaskResponse

	if taskName, ok := (param).(string); ok {
		splitName :=  strings.Split(taskName,"-")
		//Obtain workflowMap's key
		wfId, err := strconv.Atoi(splitName[1])
		if err != nil {
			panic(err.Error())
		}
		//Obtain workflowTaskMap's key
		taskNum, err := strconv.Atoi(splitName[3])
		if err != nil {
			panic(err.Error())
		}
		//Obtain the task data
		task := workflowMap[uint32(wfId)][uint32(taskNum)]

		taskPod, err := podLister.Pods(task.WorkflowId).Get(taskName)
		if err != nil {
			panic(err.Error())
		}
		taskPodHostIp := taskPod.Status.HostIP
		err = clientset.CoreV1().Pods(task.WorkflowId).Delete(taskName, &metav1.DeleteOptions{})
		deletedPodNum++
		log.Printf("Deleting taskPodName: %s.\n", taskName)
		log.Printf("--------------Deleting taskPodName %s in time:%vms on Cluster Node: %v.\n",
			taskName,time.Now().UnixNano()/1e6, taskPodHostIp)
		//write experimental data into /home/exp.txt.
		outData := "Delete " + taskName + " : " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + " on Cluster Node: "+ taskPodHostIp +"\n"
		experimentalDataObj.WriteString(outData)

		log.Printf("This is the %dth deleted workflow task pod in %v on Cluster node: %v.\n", deletedPodNum,
			task.WorkflowId, taskPodHostIp)
		//log.Println(len(workflowTaskMap))

		if task.TaskOrder == uint32(len(workflowMap[uint32(wfId)])-1) {
			//log.Println(workflowMap[uint32(wfId)])
			err := DeleteWorkflowNamespace(task)
			if err != nil {
				panic(err.Error())
			}
			taskPodResponse.Result = 1
			return &taskPodResponse, nil
			//workflowTaskMap = make(map[uint32] WorkflowTask)
		} else {
			/*The previous task has been completed and deleted, and the completed status is set to True.*/
			split := strings.Split(task.WorkflowId, "-")
			wfIdNum, err := strconv.Atoi(split[1])
			if err != nil {
				panic(err.Error())
			}
			wfTaskCompletedStateMap[uint32(wfIdNum)][task.TaskOrder] = true
			//taskCompletedStateMap[task.TaskOrder] = true
			log.Printf("The bool val of current task[%d] is: %v.\n", task.TaskOrder,
				wfTaskCompletedStateMap[uint32(wfIdNum)][task.TaskOrder])
			log.Printf("Staring to trigger the next task container.\n")
			//Trigger a subsequent task
			event.CallEvent("CreateNextTaskContainer", task.TaskName)
			taskPodResponse.Result = 1
			return &taskPodResponse, nil
		}
	} else {
		taskPodResponse.Result = 0
		return &taskPodResponse, nil
	}
}

func recoverDeleteCurrentFailedTaskContainer() {
	if r := recover(); r!= nil {
		log.Println("recovered from DeleteCurrentFailedTaskContainer()", r)
	}
}

//Delete current Failed task pod
func DeleteCurrentFailedTaskContainer(param interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error) {
	defer recoverDeleteCurrentFailedTaskContainer()
	var taskPodResponse TaskContainerBuilder.InputWorkflowTaskResponse
	if taskName, ok := (param).(string); ok {
		//task := workflowTaskMap[order]
		//taskName := task.TaskName
		splitName :=  strings.Split(taskName,"-")
		//Obtain workflow namespace's name
		//Obtain workflowMap's key
		wfId, err := strconv.Atoi(splitName[1])
		if err != nil {
			panic(err.Error())
		}
		//Obtain workflowTaskMap's key
		taskNum, err := strconv.Atoi(splitName[3])
		if err != nil {
			panic(err.Error())
		}
		//Obtain the task data
		task := workflowMap[uint32(wfId)][uint32(taskNum)]

		taskPod, err := podLister.Pods(task.WorkflowId).Get(taskName)
		if err != nil {
			panic(err.Error())
		}
		taskPodHostIp := taskPod.Status.HostIP

		err = clientset.CoreV1().Pods(task.WorkflowId).Delete(taskName, &metav1.DeleteOptions{})
		deletedPodNum++
		log.Printf("Deleting Failed taskPodName: %s.\n", taskName)
		log.Printf("--------------Deleting Failed taskPodName %s in time:%vms.\n", taskName,time.Now().UnixNano()/1e6)
		log.Printf("This is the %dth deleted workflow task pod in %v.\n", deletedPodNum, task.WorkflowId)
		if err != nil {
			panic(err.Error())
		}

		//write experimental data into /home/exp.txt.
		outData := "DeleteFailedTaskPod:" + taskName + " : " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + " on Cluster Node: "+ taskPodHostIp +"\n"
		experimentalDataObj.WriteString(outData)

		/*The Failed task has been completed and deleted, and its status is set to False*/
		wfTaskCompletedStateMap[uint32(wfId)][task.TaskOrder] = false
		log.Printf("The bool val of current task[%d] is: %v.\n",task.TaskOrder,
			wfTaskCompletedStateMap[uint32(wfId)][task.TaskOrder] )
		//log.Printf("Staring to trigger the current task container again.\n")
		////The current Failed task is triggered again.
		//event.CallEvent("AgainCreateCurrentTaskContainer",task.TaskName)
		taskPodResponse.Result = 1
		return &taskPodResponse, nil
	} else {
		taskPodResponse.Result = 0
		return &taskPodResponse, nil
	}
}

//Start to input next workflow task
func WakeUpNextWorkflow(param interface{}) (*TaskContainerBuilder.InputWorkflowTaskResponse,error)  {
	var response TaskContainerBuilder.InputWorkflowTaskResponse
	/*Obtain the serviceName of workflow injection module*/
	WorkflowInjectorServer := os.Getenv("WORKFLOW_INJECTOR_SERVICE_HOST")
	WorkflowInjectorPort := os.Getenv("WORKFLOW_INJECTOR_SERVICE_PORT")
	WorkflowInjectorServerIp := WorkflowInjectorServer + ":" + WorkflowInjectorPort
	//WorkflowInjectorServerIp = "192.168.6.111:7070"
	log.Println(WorkflowInjectorServerIp)
	if wfIndex, ok := (param).(uint32); ok {
		//Dial and connect
		log.Println("Dial Workflow Injector...")
		//WorkflowInjectorServerIp :="192.168.6.111:7070"
		conn, err := grpc.Dial(WorkflowInjectorServerIp, grpc.WithInsecure())
		if err != nil {
			panic(err.Error())
		}
		defer conn.Close()
		//Create client instance
		visitWorkflowInjectorClient := TaskContainerBuilder.NewWorkflowInjectorServiceClient(conn)
		//FinishedWorkflowId:string(strconv.Atoi(task.WorkflowId)+1)
		responseInfo := &TaskContainerBuilder.NextWorkflowSendRequest{
			FinishedWorkflowId: strconv.Itoa(int(wfIndex)),
		}
		requestNextWorkflowResponse, err := visitWorkflowInjectorClient.NextWorkflowSend(context.Background(), responseInfo)
		if err != nil {
			panic(err.Error())
		}
		if requestNextWorkflowResponse.Result == true{
			log.Println(requestNextWorkflowResponse.Result)
			log.Println("--------------The next workflow is injected at time:",time.Now().UnixNano()/1e6)
			//write experimental data into /home/exp.txt.
			outData := "Injecting workflow" + strconv.Itoa(int(wfIndex)+1)+"is over"+ ": " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) +"\n"
			experimentalDataObj.WriteString(outData)

		} else {
			log.Println("Task container builder's work is over.")
		}
		response.Result = 1
		return &response, nil
	} else {
		response.Result = 0
		return &response, nil
	}
}

/*Invoke ScheduleNextBatchTaskContainer()
Check the OutputVector of the currently completed task. If it is empty, it will be the last task in the workflow
and trigger the next workflow.
If not empty, the function fetches the task sequence number and calls the pod builder concurrently.*/
func ScheduleNextBatchTaskContainer(param interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error) {
	var response TaskContainerBuilder.InputWorkflowTaskResponse
	wait := sync.WaitGroup{}
	log.Println("Enter ScheduleNextBatchTaskContainer function...")
	if taskName, ok := (param).(string); ok {

		splitName :=  strings.Split(taskName,"-")
		//Obtain workflow namespace's name
		//Obtain workflowMap's key
		wfId, err := strconv.Atoi(splitName[1])
		if err != nil {
			panic(err.Error())
		}
		//Obtain workflowTaskMap's key
		taskNum, err := strconv.Atoi(splitName[3])
		if err != nil {
			panic(err.Error())
		}

		if task, ok := workflowMap[uint32(wfId)][uint32(taskNum)]; ok {
		//Ascending Sorting (Shortest Task First)
		task = Sorting(task,workflowMap[uint32(wfId)])
		for _, output := range task.OutputVector {
				index, err := strconv.Atoi(output)
				if err != nil {
					panic(err.Error())
				}
				log.Printf("The outVector of current task is: [%d]\n",index)
				log.Println("--------------Start time for goroutine in CreateTaskForScheduleNextBatch in time:", time.Now().UnixNano()/1e6)
				wait.Add(1)
				//To ensure that task containers are created in order, we discard go statements here
				CreateTaskForScheduleNextBatch(uint32(wfId),uint32(index),&wait)
			}
//			defer runtime.HandleCrash()
			wait.Wait()
			response.Result = 1
			return &response, nil
		} else{
			response.Result = 1
			response.VolumePath = ""
			response.ErrNo = 0
			return &response, nil
		}
	}else{
		response.Result = 1
		response.VolumePath = ""
		response.ErrNo = 0
		return &response, nil
	}
}
//Sorting Algorithms, including STF and LTF two ways
func Sorting(taskIn WorkflowTask, wfTaskMap WorkflowTaskMap) (task WorkflowTask) {

	m := map[string]uint64{}
	for _, outputValue := range taskIn.OutputVector {
		taskNum, err := strconv.Atoi(outputValue)
		if err != nil {
			panic(err.Error())
		}
		m[outputValue] = wfTaskMap[uint32(taskNum)].Duration
	}
	type Pair struct {
		Key   string
		Value uint64
	}
	pairs := make([]Pair, len(m))
	i := 0
	for k, v := range m {
		pairs[i] = Pair{Key: k, Value: v}
		i++
	}
	//Sort in ascending order, Shortest Task First
	//sort.Slice(pairs, func(i, j int) bool {
	//	return pairs[i].Value < pairs[j].Value
	//})

	//Sort in descending order, Longest Task First
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})
	// Output Printing
	taskIn.OutputVector = taskIn.OutputVector[:0]
	for _, pair := range pairs {
		taskIn.OutputVector = append(taskIn.OutputVector, pair.Key)
		log.Println("Key:", pair.Key, "Value:", pair.Value)
	}
	return taskIn
}

func CreateTaskForScheduleNextBatch(wfNum uint32,order uint32,waiter *sync.WaitGroup,) {
//func CreateTaskForScheduleNextBatch(order uint32) {

	defer waiter.Done()
	log.Println("Enter CreateTaskForScheduleNextBatch function...")
		if task, ok := workflowMap[wfNum][order]; ok {
			/*When this task is the first task of current workflow.*/
			if task.InputVector == nil {
				taskSema <- 1
				_, _, err := CreateTaskContainer(task,clientset,redisIpPort)
				<-taskSema
				if err != nil {
					panic(err.Error())
					//return  volumePath, errNo, err
				}
				//return volumePath, errNo, nil
			} else{
				/* Check InputVector*/
				accumulatedBoolVar := true
				for _, input := range task.InputVector {
					index, err := strconv.Atoi(input)
					if err != nil {
						panic(err.Error())
					}
					log.Printf("The intVector of current task is: [%d]\n",index)
					log.Printf("The bool val of task[%d] is: %v.\n",index,wfTaskCompletedStateMap[wfNum][uint32(index)])
					accumulatedBoolVar = accumulatedBoolVar && wfTaskCompletedStateMap[wfNum][uint32(index)]
				}
				log.Printf("The intVector accumulatedBoolVar of current task is: [%v]\n",accumulatedBoolVar)
				/*Create a pod for this task when all the previous tasks of the current task have completed and the
				current task have not been executed*/
				if accumulatedBoolVar && (!wfTaskCompletedStateMap[wfNum][task.TaskOrder]) {
					log.Println("Prefix task of current task are all completed.")
					log.Printf("--------------Starting to run task[%d] in time:%vms.\n",task.TaskOrder,time.Now().UnixNano()/1e6)
					taskSema <- 1
					_, _, err := CreateTaskContainer(task,clientset,redisIpPort)
					<-taskSema
					if err != nil {
						panic(err.Error())
						//return volumePath, errNo, err
					}
					//return volumePath, errNo, err
				}
				//return "", 0 ,nil
			}
		}
}
/*Create task container*/
func CreateTask(param interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error) {
	var taskPodResponse TaskContainerBuilder.InputWorkflowTaskResponse

	if taskName, ok := (param).(string); ok {
		log.Printf("Passed parameters-taskName: %v\n", taskName)
		splitName := strings.Split(taskName, "-")
		//Obtain workflow namespace's name
		//Obtain workflowMap's key
		wfId, err := strconv.Atoi(splitName[1])
		if err != nil {
			panic(err.Error())
		}
		//Obtain workflowTaskMap's key
		taskNum, err := strconv.Atoi(splitName[3])
		if err != nil {
			panic(err.Error())
		}

		//if task, ok := workflowMap[uint32(wfId)][uint32(taskNum)]; ok {
		if task, ok := workflowTaskMap[uint32(taskNum)]; ok {
			//log.Println(task)
			if task.InputVector == nil {
				taskSema <- 1
				volumePath, errNo, err := CreateTaskContainer(task, clientset, redisIpPort)
				<-taskSema

				taskPodResponse.VolumePath = volumePath
				if err != nil {
					panic(err.Error())
					taskPodResponse.ErrNo = errNo
					return &taskPodResponse, nil
				}
				taskPodResponse.Result = 0
				return &taskPodResponse, nil
			} else {
				accumulatedBoolVar := true
				for _, input := range task.InputVector {
					index, err := strconv.Atoi(input)
					if err != nil {
						panic(err.Error())
					}
					accumulatedBoolVar = accumulatedBoolVar && wfTaskCompletedStateMap[uint32(wfId)][uint32(index)]
				}
				if accumulatedBoolVar && (!wfTaskCompletedStateMap[uint32(wfId)][task.TaskOrder]) {
					log.Println("Prefix task of current task are all completed.")

					taskSema <- 1
					volumePath, errNo, err := CreateTaskContainer(task, clientset, redisIpPort)
					<- taskSema

					taskPodResponse.VolumePath = volumePath
					if err != nil {
						panic(err.Error())
						taskPodResponse.ErrNo = errNo
						return &taskPodResponse, nil
					}
					taskPodResponse.Result = 0
					return &taskPodResponse, nil

				} else {
					taskPodResponse.Result = 1
					return &taskPodResponse, nil
				}
			}

		} else {
			taskPodResponse.Result = 1
			return &taskPodResponse, nil
		}
	}else {
		taskPodResponse.Result = 1
		return &taskPodResponse, nil
	}
}

func CreateCurrentTaskAgain(param interface{})(*TaskContainerBuilder.InputWorkflowTaskResponse,error) {
	var taskPodResponse TaskContainerBuilder.InputWorkflowTaskResponse

	if taskName, ok := (param).(string); ok {
		log.Printf("Passed parameters-taskName: %v\n", taskName)
		splitName := strings.Split(taskName, "-")
		//Obtain workflow namespace's name
		//Obtain workflowMap's key
		wfId, err := strconv.Atoi(splitName[1])
		if err != nil {
			panic(err.Error())
		}
		//Obtain workflowTaskMap's key
		taskNum, err := strconv.Atoi(splitName[3])
		if err != nil {
			panic(err.Error())
		}

		if task, ok := workflowMap[uint32(wfId)][uint32(taskNum)]; ok {
		//if task, ok := workflowTaskMap[uint32(taskNum)]; ok {
			//log.Println(task)
			if task.InputVector == nil {
				taskSema <- 1
				volumePath, errNo, err := CreateTaskContainer(task, clientset, redisIpPort)
				<-taskSema

				taskPodResponse.VolumePath = volumePath
				if err != nil {
					panic(err.Error())
					taskPodResponse.ErrNo = errNo
					return &taskPodResponse, nil
				}
				taskPodResponse.Result = 0
				return &taskPodResponse, nil
			} else {
				accumulatedBoolVar := true
				for _, input := range task.InputVector {
					index, err := strconv.Atoi(input)
					if err != nil {
						panic(err.Error())
					}
					accumulatedBoolVar = accumulatedBoolVar && wfTaskCompletedStateMap[uint32(wfId)][uint32(index)]
				}
				if accumulatedBoolVar && (!wfTaskCompletedStateMap[uint32(wfId)][task.TaskOrder]) {
					log.Println("Prefix task of current task are all completed.")
					taskSema <- 1
					volumePath, errNo, err := CreateTaskContainer(task, clientset, redisIpPort)
					<-taskSema

					taskPodResponse.VolumePath = volumePath
					if err != nil {
						panic(err.Error())
						taskPodResponse.ErrNo = errNo
						return &taskPodResponse, nil
					}
					taskPodResponse.Result = 0
					return &taskPodResponse, nil

				} else {
					taskPodResponse.Result = 1
					return &taskPodResponse, nil
				}
			}

		} else {
			taskPodResponse.Result = 1
			return &taskPodResponse, nil
		}
	}else {
		taskPodResponse.Result = 1
		return &taskPodResponse, nil
	}
}


func CreateTaskContainer(task WorkflowTask,clientService *kubernetes.Clientset, redisIpPort string)(string, uint32, error) {

	//taskSema <- 1
	//clientNamespace, isFirstPod, clientPvcOfThisNamespace, err := createTaskPodNamespaces(request, clientset)
	clientNamespace, isFirstPod, clientPvcOfThisNamespace, err := createTaskPodNamespaces(task, clientService)
	if err != nil {
		panic(err.Error())
	}
	//<-taskSema

	//volumePath, err := clientTaskCreatePod(request, clientset, clientNamespace, isFirstPod, clientPvcOfThisNamespace)
	volumePath, errNo, err := clientTaskCreatePod(task, clientService, clientNamespace, isFirstPod, clientPvcOfThisNamespace)

	return  volumePath,errNo, err
}

func updateDataInRedisFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from updateDataInRedisFail()", r)
	}
}

func updateDataInRedis(task WorkflowTask, redisServer string){
	var unMarshalData saveWorkflowTaskData
	defer updateDataInRedisFail()

	// build connection
    log.Printf("redisServer: %v\n",redisServer)
	redisSema <- 1
	conn, err := redis.Dial("tcp", redisServer)
	if err != nil {
		log.Println("redis.Dial err=", err)
		return
	}

	r, err := redis.Bytes(conn.Do("get", task.TaskName))
	if err != nil {
		log.Println("get err=", err)
		return
	}

	err = json.Unmarshal(r,&unMarshalData)
	if err != nil {
		log.Println("json unMarshal is err: ", err)
		return
	}

	unMarshalData.AliveStatus = true
	//结构体序列化为字符串
	taskData, err := json.Marshal(unMarshalData)
	// 通过go向redis写入数据 string [key - value]

	_, err = conn.Do("set", task.TaskName, taskData)
	if err != nil {
		log.Println("set err=", err)
		return
	}
    log.Printf("taskName: %v,taskData: %v\n",task.TaskName, unMarshalData)
	// 关闭连接
	conn.Close()

	<- redisSema
}

func recoverNamespaceListerFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from createTaskPodNamespaces()", r)
	}
}

//Create namespace
func createTaskPodNamespaces(task WorkflowTask,clientService *kubernetes.Clientset)(*v1.Namespace, bool,
	*v1.PersistentVolumeClaim, error) {
	defer recoverNamespaceListerFail()
	name := task.WorkflowId
	namespacesClient := clientService.CoreV1().Namespaces()
	/*3.1 Monitor Namespace resources and obtain namespaceLister using the Informer tool package*/
	namespaceList, err := namespaceLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}
	namespaceExist := false
	for _, ns := range namespaceList{
		if ns.Name == name {
			namespaceExist = true
			break
		}
	}
	// Create pvc's client
	pvcClient := clientService.CoreV1().PersistentVolumeClaims(name)
	/*When the Namespace of this workflow exists...*/
	if namespaceExist {
		nsClientObject ,err := namespacesClient.Get(name,metav1.GetOptions{})
		if err != nil {
			panic(err)
		}
		log.Printf("This namespace %v is already exist.\n", name)
		pvcObjectAlreadyInThisNamespace, err := pvcClient.Get(name+"-pvc", metav1.GetOptions{})
		//The namespace is already exist and there have been a pod in this namespace, podIsFirstInThisNamespace
		//condition is false
		podIsFirstInThisNamespace := false
		return nsClientObject, podIsFirstInThisNamespace, pvcObjectAlreadyInThisNamespace,nil
	}else{
		taskNsNum++
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{"namespace": "task"},
			},
			Status: v1.NamespaceStatus{
				Phase: v1.NamespaceActive,
			},
		}
		//write experimental data into /home/exp.txt.
		outData := "Create" + name + ": " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + "\n"
		experimentalDataObj.WriteString(outData)
		// Create a new Namespaces
		clientNamespaceObject, err := namespacesClient.Create(namespace)
		podIsFirstInThisNamespace := true
		if err != nil {
			panic(err)
		}
		log.Printf("Creating Namespace %s\n", clientNamespaceObject.ObjectMeta.Name)
		log.Printf("Creating Namespaces...,this is %dth namespace.\n",taskNsNum)
		//Create namespace's pvc

		pvcObjectInThisNamespace,err := createPvc(clientService,name)
		if err != nil {
			panic(err)
		}
		return clientNamespaceObject,podIsFirstInThisNamespace, pvcObjectInThisNamespace, nil
		//return clientNamespaceObject,podIsFirstInThisNamespace, nil
	}
}
func recoverCreatePvcFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from ", r)
	}
}
func createPvc(clientService *kubernetes.Clientset,nameOfNamespace string)(*v1.PersistentVolumeClaim,error) {
	defer recoverCreatePvcFail()
	storageClassName := "nfs-storage"
	pvcClient := clientService.CoreV1().PersistentVolumeClaims(nameOfNamespace)
	pvc := new(v1.PersistentVolumeClaim)
	pvc.TypeMeta = metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"}
	pvc.ObjectMeta = metav1.ObjectMeta{ Name: nameOfNamespace +"-pvc",
	//Create namespace's pvc
		Labels: map[string]string{"pvc": "nfs"},
		Namespace: nameOfNamespace }

		pvc.Spec = v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteMany,
			},
			Resources: v1.ResourceRequirements{
				Requests:v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("10" + "Mi"),
				},
			},
			//Selector: &metav1.LabelSelector{
			//	MatchLabels:
			//	map[string]string{"app": "nfs"},
			//},
			StorageClassName: &storageClassName,
		}
	//}
	//write experimental data into /home/exp.txt.
	outData := "Create " + pvc.Name + ": " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + "\n"
	experimentalDataObj.WriteString(outData)

	pvcResult, err := pvcClient.Create(pvc)
	if err != nil {
		panic(err)
	}
	log.Printf("Created Pvc %s on %s\n", pvcResult.ObjectMeta.Name,
		pvcResult.ObjectMeta.CreationTimestamp)

	return pvcResult, nil
}

func recoverCreatePodFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from recoverCreatePodFail()", r)
	}
}
//4. Task container creator module
func clientTaskCreatePod(task WorkflowTask ,clientService *kubernetes.Clientset,
	podNamespace *v1.Namespace, IsfirstPod bool, pvcClient *v1.PersistentVolumeClaim)(string, uint32, error) {
	defer recoverCreatePodFail()
	//schedulerName := "default-scheduler"
	taskPodNum++
	//taskPodName := task.WorkflowId + "-task-"+strconv.Itoa(int(task.TaskOrder))
	taskPodName := task.TaskName

	taskPod, err := podLister.Pods(task.WorkflowId).Get(taskPodName)
	if err == nil {
		thisTaskPodExist = true
		log.Printf("This task pod: %v is already exist in %v.\n", taskPod.Name, taskPod.Namespace)
	} else {
		//panic(err)
		/*err不为nil,说明Informer工具包的podLister不能获取到，此任务pod不存在*/
		thisTaskPodExist = false
	}

	if thisTaskPodExist == true {
		volumePathInContainer := "/nfs/data"
		event.CallEvent("DeleteCurrentTaskContainer", task.TaskName)
		returnErrNo := uint32(1)
		return volumePathInContainer, returnErrNo, nil

	} else {
		//Cloud-edge resource evaluation algorithm
		task.Cpu, task.Mem = CloudEdgeResourceEvaluationBaseAlgorithm(task,redisIpPort)
		//task.Cpu, task.Mem = CloudEdgeResourceScalingAlgorithm(task,redisIpPort)
		//write experimental data into /home/exp.txt.
		outData := "AllocationResource: " + task.TaskName + " : " + "CPU: " + strconv.Itoa(int(task.Cpu)) + ": Memory: "+ strconv.Itoa(int(task.Mem)) +"\n"
		experimentalDataObj.WriteString(outData)

		log.Printf("resourceAllocate: %dm, %dMi.\n",task.Cpu,task.Mem)

		//Update task's completed time in Redis
		updateDataInRedis(task,redisIpPort)

		path, returnErrNo, err := CreateTaskPod(task, clientService, podNamespace, IsfirstPod, pvcClient)
		VolumePath := path
		if err != nil {
			panic(err.Error())
		}
		return VolumePath, returnErrNo, nil
	}
}
//Cloud-edge: Baseline--As long as edge node can accommodate sufficient resources ,
//the Baseline algorithm allocate resources, otherwise it waits until the resources are released.
func CloudEdgeResourceEvaluationBaseAlgorithm(task WorkflowTask,redisIpPort string) (cpuNum uint64, memNum uint64) {
	//When the remaining resource suffices for request resource of task Pod, break the for loop.
	currentNodeResidualCpuResource := uint64(0)
	currentNodeResidualMemResource := uint64(0)
	for {
		//Obtain the map of remaining resources of each node
		ResidualMap := k8sResource.GetK8sEachNodeResource(podLister, nodeLister, nodeResidualMap)
		//Obtain the remaining resources in the current node with this task
		for _, labelValue := range task.Labels {
			for nodeIp, val := range ResidualMap {
				if nodeIp == labelValue {
					log.Println(task.Labels)
					//if (val.MilliCpu > nodeResidualResourceMaxCpu) && (val.MilliCpu != 0) {
					//	nodeResidualResourceMaxCpu = val.MilliCpu
					//	nodeResidualResourceMaxMem = val.Memory
					//}
					currentNodeResidualCpuResource = val.MilliCpu
					currentNodeResidualMemResource = val.Memory
					break
				}
			}
		}
		log.Printf("currentNodeResidualCpuResource: %v,currentNodeResidualMemResource: %v\n",
			currentNodeResidualCpuResource, currentNodeResidualMemResource)
		/*If the maximum remaining resources of a cluster node are greater than the resources required by the current
		  task, a task container would be generated*/
		if (currentNodeResidualCpuResource >= task.Cpu) && (currentNodeResidualMemResource >= task.Mem) {
			//Create this task POD if the cluster resources are sufficient
			break
		}
		//log.Println("Remaining resources fail to suffice for creating task pods and execute for loop.")
	}
	return task.Cpu, task.Mem
}

//resource allocation algorithm
//It is designed to allocate resource through customized algorithm which is integrated in modular manner.
func CloudEdgeResourceScalingAlgorithm(task WorkflowTask,redisIpPort string) (cpuNum uint64, memNum uint64) {

	//Get resource quota for the current task pod through for loop.
  for {
	  //First, connect redis Key-value database and read resource data of tasks to be launched recently.
	  accumulatedCpuFromRedis, accumulatedMemFromRedis := acquireRelatedDataFromRedis(task, redisIpPort)

	  log.Printf("accumulatedCpuFromRedisSameNode: %v,accumulatedMemFromRedisSameNode: %v\n",
		  accumulatedCpuFromRedis, accumulatedMemFromRedis)

	  currentNodeResidualCpuResource := uint64(0)
	  currentNodeResidualMemResource := uint64(0)
	  //nodeResidualResourceMaxCpu := uint64(0)
	  //nodeResidualResourceMaxMem := uint64(0)
	  //Obtain the map of remaining resources of each node
	  ResidualMap := k8sResource.GetK8sEachNodeResource(podLister, nodeLister, nodeResidualMap)

	  //Obtain the remaining resources in the current node with this task
	  for _, labelValue := range task.Labels {
		  for nodeIp, val := range ResidualMap {
			  if nodeIp == labelValue {
				  log.Println(task.Labels)
				  currentNodeResidualCpuResource = val.MilliCpu
				  currentNodeResidualCpuResource = val.Memory
				  break
			  }
		  }
	  }
	  log.Printf("cpuResidualResourceValue: %v,memResidualResourceValue: %v\n",
		  currentNodeResidualCpuResource, currentNodeResidualMemResource)

	  if (accumulatedCpuFromRedis != 0) && (accumulatedMemFromRedis != 0) {
		  //(1)If the maximum remaining resources of the current node are greater than the resources required by all the concurrent
		  //     task, a task container can be generated*/
		  if (accumulatedCpuFromRedis <= currentNodeResidualCpuResource) &&
			  (accumulatedMemFromRedis <= currentNodeResidualMemResource) {
				  cpuNum = task.Cpu
				  memNum = task.Mem
				  log.Printf("allocation pod's cpuNum: %dm, allocation pod's memNum: %dMi.\n", cpuNum, memNum)
				  return cpuNum, memNum
		  }
		  //(2) The remaining CPU resource of the current node is insufficient.
		  //     Compress the CPU resources of tasks in proportion.
		  if (accumulatedCpuFromRedis > currentNodeResidualCpuResource) &&
			  (accumulatedMemFromRedis < currentNodeResidualMemResource) {
				  //compress the Cpu's requests resource
				  cpuNumTemp := task.Cpu * currentNodeResidualCpuResource / accumulatedCpuFromRedis
				  cpuNum = cpuNumTemp
				  memNum = task.Mem
				  log.Printf("allocation pod's cpuNum: %dm, allocation pod's memNum: %dMi.\n", cpuNum, memNum)
				  return cpuNum, memNum
			  }
		  //(3)The remaining MEM resource of the task is insufficient.
		  //Compress the MEM resources of tasks in proportion.
		  if (accumulatedCpuFromRedis < currentNodeResidualCpuResource) &&
			  (accumulatedMemFromRedis > currentNodeResidualMemResource) {
			  //compress the memory's requests resource
			  cpuNum = task.Cpu
			  memNumTemp := task.Mem * currentNodeResidualMemResource / accumulatedMemFromRedis
			  memNum = memNumTemp
			  log.Printf("allocation pod's cpuNum: %dm, allocation pod's memNum: %dMi.\n", cpuNum, memNum)
			  return cpuNum, memNum
			  }

			//(4) Both remaining resources are insufficient.
			//Compress the CPU and MEM resources of tasks in proportion.
		  if (accumulatedCpuFromRedis >= currentNodeResidualCpuResource) &&
			  (accumulatedMemFromRedis >= currentNodeResidualMemResource) {
			  cpuNumTemp := task.Cpu * currentNodeResidualCpuResource / accumulatedCpuFromRedis
			  memNumTemp := task.Mem * currentNodeResidualMemResource / accumulatedMemFromRedis
			  cpuNum = cpuNumTemp
			  memNum = memNumTemp
			  log.Printf("allocation pod's cpuNum: %dm, allocation pod's memNum: %dMi.\n", cpuNum, memNum)
		  }
	  } else {
		  //failed to acquire resource data from redis.
		  if task.Cpu < currentNodeResidualCpuResource && task.Mem < currentNodeResidualMemResource {
			  cpuNum = task.Cpu
			  memNum = task.Mem
			  log.Printf("获取redis总的资源总任务资源为0，没有链接上，适当非配资源为requests,allocation cpuNum: %dm, "+
				  "allocation memNum: %dMi.\n", cpuNum, memNum)
			  //return cpuNum, memNum
		  } else {
			  if task.Cpu >= currentNodeResidualCpuResource && task.Mem < currentNodeResidualMemResource {
				  cpuNum = currentNodeResidualCpuResource * 8 / 10
				  memNum = task.Mem
				  log.Printf("获取redis总的资源总任务资源为0，没有链接上，适当非配资源为requests,allocation cpuNum: %dm, "+
					  "allocation memNum: %dMi.\n", cpuNum, memNum)
				  //return cpuNum, memNum
			  } else {
				  if task.Cpu < currentNodeResidualCpuResource && task.Mem >= currentNodeResidualMemResource {
					  cpuNum = task.Cpu
					  memNum = currentNodeResidualCpuResource * 8 / 10
					  log.Printf("获取redis总的资源总任务资源为0，没有链接上，适当非配资源为requests,allocation cpuNum: %dm, "+
						  "allocation memNum: %dMi.\n", cpuNum, memNum)
					  //return cpuNum, memNum
				  } else {
					  cpuNum = currentNodeResidualCpuResource * 8 / 10
					  memNum = currentNodeResidualMemResource * 8 / 10
					  log.Printf("获取redis总的资源总任务资源为0，没有链接上，适当非配资源为requests,allocation cpuNum: %dm, "+
						  "allocation memNum: %dMi.\n", cpuNum, memNum)
					  //return cpuNum, memNum
				  }
			  }
		  }
	  }
	  //Check whether the assigned value meets the resource limit of the task container
	  //stress -m 100 (require ) In experiments, the container requires 20 Mi more memory
	  //than allocated memory by stress.
	  //CPU resources are compressible and unlimited.
	  //To ensure that the container works properly, we set the minimum value.
	  if cpuNum >= task.MinCpu && memNum >= (task.MinMem + 20) {
	  	  break
	  }
  }
	return cpuNum, memNum
}

func acquireRelatedDataFromRedisFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from acquireRelatedDataFromRedisFail()", r)
	}
}
func acquireRelatedDataFromRedis(task WorkflowTask, redisServer string)(totalCpu uint64, totalMem uint64) {
	var readDataFromRedis saveWorkflowTaskData
	var currentTaskDataFromRedis saveWorkflowTaskData
	defer acquireRelatedDataFromRedisFail()

	log.Printf("redisServer: %v\n",redisServer)
	// build connection
	conn, err := redis.Dial("tcp", redisServer)
	if err != nil {
		log.Println("redis.Dial err=", err)
		return
	}
	//get the deadline of the current task to be created.

	r, err := redis.Bytes(conn.Do("get", task.TaskName))
	if err != nil {
		log.Println("get err=", err)
		return
	}

	//deserialization
	err = json.Unmarshal(r,&currentTaskDataFromRedis)
	if err != nil {
		log.Println("json unMarshal is err: ", err)
		return
	}
	totalCpu = currentTaskDataFromRedis.MilliCpu
	totalMem = currentTaskDataFromRedis.Memory
	log.Printf("currentTaskMilliCpu: %v,currentTaskMemory: %v\n",currentTaskDataFromRedis.MilliCpu,
		currentTaskDataFromRedis.Memory)
	// read task data from redis
    //Firstly, obtain all keys from redis.
	cacheName := "workflow*"
	keys,err := redis.Strings(conn.Do("keys", cacheName))

    //Second, get the corresponding values by using key.
	for _,key := range keys {
		taskData,err := redis.Bytes(conn.Do("get", key))
		if err != nil {
			log.Println("get err=", err)
			return
		}
        //deserialization
		err = json.Unmarshal(taskData,&readDataFromRedis)
		if err != nil {
			log.Println("json unMarshal is err: ", err)
			return
		}
		//Check the AliveStatus and compare deadline
		for taskLabelKey, _ := range task.Labels {
			if readDataFromRedis.AliveStatus == false &&
				readDataFromRedis.Labels[taskLabelKey] == currentTaskDataFromRedis.Labels[taskLabelKey] {
				if readDataFromRedis.StartTime < currentTaskDataFromRedis.Deadline &&
					readDataFromRedis.StartTime >= currentTaskDataFromRedis.StartTime {
					totalCpu += readDataFromRedis.MilliCpu
					totalMem += readDataFromRedis.Memory
				}
			}
		}
		//log.Printf("%v: %v\n",key, readDataFromRedis)
	}
	// 关闭连接
	conn.Close()

    return totalCpu, totalMem
}

func CreateTaskPod(task WorkflowTask ,clientService *kubernetes.Clientset,
	podNamespace *v1.Namespace, IsfirstPod bool, pvcClient *v1.PersistentVolumeClaim)(string, uint32, error){

	//if IsfirstPod {
	//	//hostPath := "/nfs/data/"
	//}
	pod := new(v1.Pod)
	//pod.TypeMeta = unversioned.TypeMeta{Kind: "Pod", APIVersion: "v1"}
	pod.TypeMeta =  metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}
	pod.ObjectMeta = metav1.ObjectMeta{
		//Name: task.WorkflowId + "-" + strconv.Itoa(int(task.TaskOrder)),
		Name: task.TaskName,
		Namespace: podNamespace.Name,
		Labels: map[string]string{"app": "task"},
		Annotations: map[string]string{"AnnotationsName": "task pods of workflow."}}
	volumePathInContainer := "/nfs/data"

	pod.Spec = v1.PodSpec{

		RestartPolicy: v1.RestartPolicyNever,
		SchedulerName: "default-scheduler",
		NodeSelector: task.Labels,
		Volumes: []v1.Volume{
			v1.Volume{
				Name: "pod-share-volume",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcClient.ObjectMeta.Name,
					},
				},
			},
		},
		Containers: []v1.Container{
			v1.Container{
				Name:    "task-pod",
				Image:   task.Image,
				Command: nil,
				//Command: nil,
				//Args:[]string{"-c","1","-m","100","-t","5","-i","3"},
				Args: task.Args,
				Ports: []v1.ContainerPort{
					v1.ContainerPort{
						ContainerPort: 80,
						Protocol:      v1.ProtocolTCP,
					},
				},
				VolumeMounts: []v1.VolumeMount {
					v1.VolumeMount{
						Name: "pod-share-volume",
						MountPath: volumePathInContainer,
						//SubPath: pvcClient.ObjectMeta.Name,
					},
				},
				Env: []v1.EnvVar{
					v1.EnvVar{
						Name:  "VOLUME_PATH",
						Value: volumePathInContainer,
						//ValueFrom: &v1.EnvVarSource{
						//	FieldRef: &v1.ObjectFieldSelector{
						//		FieldPath: volumePathInContainer,
						//	},
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(strconv.Itoa(int(task.Cpu)) + "m"),
						v1.ResourceMemory: resource.MustParse(strconv.Itoa(int(task.Mem)) + "Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(strconv.Itoa(int(task.Cpu)) + "m"),
						v1.ResourceMemory: resource.MustParse(strconv.Itoa(int(task.Mem)) + "Mi"),
					},
				},
				ImagePullPolicy: v1.PullIfNotPresent,
				//ImagePullPolicy: v1.PullAlways,
			},
			},
		}
	//write experimental data into /home/exp.txt.
	outData := "CreateTaskPod " + pod.Name + ": " + strconv.Itoa(int(time.Now().UnixNano()/1e6)) + "\n"
	experimentalDataObj.WriteString(outData)

	_, err := clientService.CoreV1().Pods(podNamespace.Name).Create(pod)
	if err != nil {
		panic(err.Error())
	}
	//namespace := podNamespace.Name
	pods, err := clientService.CoreV1().Pods(podNamespace.Name).List(metav1.ListOptions{})
	if err != nil {
		panic(err)
		//return "", err
	}
	log.Printf("There are %d pods in current namespace:%s\n", len(pods.Items), task.WorkflowId)
	log.Printf("This is %dth task pod in all workflow namespaces.\n", taskPodNum)
	//for _, pod := range pods.Items {
	//	log.Printf("Name: %s, Status: %s, CreateTime: %s\n", pod.ObjectMeta.Name, pod.Status.Phase, pod.ObjectMeta.CreationTimestamp)
	//}
	errNo := uint32(0)
	return volumePathInContainer,errNo, nil
}

func taskBuilderServer(waiter *sync.WaitGroup) {
	defer waiter.Done()
	//Build grpc server
	server := grpc.NewServer()
	log.Println("Build workflow task builder gRPC Server.")
	//Register the resource request service
	TaskContainerBuilder.RegisterTaskContainerBuilderServiceServer(server,new(ResourceServiceImpl))
	//Listen on 7070 port
	lis, err := net.Listen("tcp", ":7070")
	log.Println("Listening local port 7070.")
	if err != nil {
		panic(err.Error())
	}
	server.Serve(lis)
}

//Initialize the nodeResidualResourceMap
func initNodeResidualResourceMap(resourceMap k8sResource.ResidualResourceMap, clusterMasterIp string) k8sResource.ResidualResourceMap {
	//nodeIpPrefix := "121.250.173."
	splitName :=  strings.Split(clusterMasterIp,".")
	nodeIpFourthField, err := strconv.Atoi(splitName[len(splitName)-1])
	if err != nil {
		panic(err)
	}
	nodeNum, err := strconv.Atoi(os.Getenv("NODE_NUM"))
	if err != nil {
		panic(err)
	}
	nodeIpThirdField, err := strconv.Atoi(splitName[2])
	if err != nil {
		panic(err)
	}
	nodeIpPrefix := splitName[0] + "." +splitName[1] + "." + splitName[2] + "."

	for i := 1; i <= nodeNum; i++ {
		if (nodeIpFourthField+i) < 256 {
			nodeResidualResourceKey := nodeIpPrefix + strconv.Itoa( nodeIpFourthField+i )
			resourceMap[nodeResidualResourceKey] = k8sResource.NodeResidualResource{0, 0}
		}else {
			nodeIpFourthField = nodeIpFourthField + i - 256
			nodeIpThirdField = nodeIpThirdField + 1
			nodeIpPrefix = splitName[0] + "." +splitName[1] + "." + strconv.Itoa(nodeIpThirdField) + "."
			nodeResidualResourceKey := nodeIpPrefix + strconv.Itoa(nodeIpFourthField+i)
			resourceMap[nodeResidualResourceKey] = k8sResource.NodeResidualResource{0, 0}
		}
	}
	log.Println(resourceMap)
	return resourceMap
}
//Event trigger module
func eventRegister(){
	// Register event
	event.RegisterEvent("CreateNextTaskContainer", ScheduleNextBatchTaskContainer)
	event.RegisterEvent("AgainCreateCurrentTaskContainer",CreateCurrentTaskAgain)
	event.RegisterEvent("ThisWorkflowEnd", WakeUpNextWorkflow)
	event.RegisterEvent("DeleteCurrentTaskContainer", DeleteCurrentTaskContainer)
	event.RegisterEvent("DeleteCurrentFailedTaskContainer",DeleteCurrentFailedTaskContainer)
}
func main() {
	taskCompletedStateMap = make(map[uint32]bool)
	logFile, err := os.OpenFile("/home/log.txt", os.O_CREATE | os.O_APPEND | os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	mw := io.MultiWriter(os.Stdout,logFile)
	log.SetOutput(mw)
	//create experimental data file
	experimentalDataObj, _ = os.OpenFile("/home/exp.txt", os.O_CREATE | os.O_APPEND | os.O_RDWR, 0666)
	defer experimentalDataObj.Close()

	//Get MasterIp by env
	masterIp = os.Getenv("MASTER_IP")
	log.Printf("masterIp: %v\n",masterIp)
	//Get time interval of sample by env
	gatherTime = os.Getenv("GATHER_TIME")
	log.Printf("gatherTime: %v\n",gatherTime)
	valTime, err := strconv.Atoi(gatherTime)
	if err != nil {
		panic(err)
	}
	interval = uint32(valTime)
	//Acquire the env of Redis's Ip
	RedisServer := os.Getenv("REDIS_SERVER")
	RedisServerPort := os.Getenv("REDIS_PORT")
	redisIpPort = RedisServer + ":" + RedisServerPort
	log.Println(redisIpPort)

	//Create chan for Informer
	stopper := make(chan struct{})
	defer close(stopper)
    /*Create a map of remaining resources for each node*/
	nodeResidualResourceMap = make(k8sResource.ResidualResourceMap)
	nodeResidualMap = initNodeResidualResourceMap(nodeResidualResourceMap, masterIp)

	waiter := sync.WaitGroup{}
	waiter.Add(1)

	//Start gRPC Server
	go taskBuilderServer(&waiter)
	//Start event register
	eventRegister()
	//Create K8s's client
	clientset = k8sResource.GetRemoteK8sClient()
	podLister, nodeLister, namespaceLister = informer.InitInformer(stopper,"/kubelet.kubeconfig")

	//Collect allocatable and request resources periodically
	//go gatherResource(&waiter,allocateResourceMap,usedResourceMap,interval)

	//defer runtime.HandleCrash()

	waiter.Wait()
}
