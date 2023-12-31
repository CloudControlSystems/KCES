package main

import (
	"WorkflowInjector/messageProto/TaskContainerBuilder"
	context2 "context"
	"encoding/json"
	_ "fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
	"github.com/gomodule/redigo/redis"
)

//var ch = make(chan int)
//Define workflow structure
type WorkflowTask struct {
	//workflow的ID
	WorkflowId string
	//taskNum
	TaskNum uint32
	//taskName
	TaskName string
	//task Image
	Image string
	//millicore(1Core=1000millicore)
	Cpu uint64
	//MiB
	Mem uint64
	//task order
	TaskOrder uint32
	//env
	Env map[string]string
	// Input Vector
	InputVector []string
	// Out Vector
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
//var workflowTask WorkflowTask
//var wg *sync.WaitGroup
var taskContainerBuilderServerIp string
var redisIpPort string
type workflowTaskMap map[uint64]WorkflowTask
var workflowMap map[uint64]map[uint64]WorkflowTask
var wfTaskMap map[uint64]WorkflowTask
//var dependencyMap map[uint32]map[string][]string
var redisSema = make(chan int, 1)
var taskPackageSema = make(chan int, 1)
var wfSema = make(chan int, 1)
type saveWorkflowTaskData struct {
	StartTime uint64
	Duration uint64
	Deadline uint64
	AliveStatus bool
	MilliCpu uint64
	Memory uint64
}

type ResourceServiceImpl struct {

}
/*Start the next workflow injection request from TaskContainerBuilder*/
func (r ResourceServiceImpl) NextWorkflowSend(ctx context2.Context, request *TaskContainerBuilder.NextWorkflowSendRequest) (*TaskContainerBuilder.NextWorkflowSendResponse, error) {
	var response  TaskContainerBuilder.NextWorkflowSendResponse
	//taskContainerBuilderServerIp = "192.168.6.110:7070"
	log.Println("Receive creating next workflow information from TaskContainerBuilder.")
	finishWfTaskId := request.FinishedWorkflowId
	id, err := strconv.Atoi(finishWfTaskId)
	if err != nil {
		panic(err.Error())
	}
	//log.Println(workflowMap)
	nextWorkflowId := id + 1
	/*Determine whether workflow input is complete.*/
	if nextWorkflowId <= len(workflowMap) - 1 {
		nextWfTaskMap := workflowMap[uint64(nextWorkflowId)]
		log.Println(nextWfTaskMap)
		visitRemoteTaskContainerBuilder(taskContainerBuilderServerIp,nextWfTaskMap)
		response.Result = true
		return &response, nil
	} else {
		response.Result = false
		log.Println("workflow injecting is over.")
		return &response, nil
	}
}
func visitRemoteTaskContainerBuilder(serverIp string,wf workflowTaskMap) {
	//1.Dial
	log.Println("Testing workflow task container...")

	RedisServer := os.Getenv("REDIS_SERVER")
	RedisServerPort := os.Getenv("REDIS_PORT")
	redisServer := RedisServer + ":" + RedisServerPort

	//Send workflow following task's sequence.
	for i:= 0; i < len(wf); i++ {
	//for _, val := range wf {
		val := wf[uint64(i)]

		log.Printf("sending task: %v\n",val)
		//write Key-value data (TaskName:{startTime,duration,deadline,Cpu,Mem,InputVector,OutputVector})
		log.Printf("redisServer: %v\n",redisServer)

		//writeDataInRedis(val,redisServer)

		conn, err := grpc.Dial(serverIp, grpc.WithInsecure())
		log.Println(serverIp)
		if err != nil {
			panic(err.Error())
		}
		//Create client instance
		visitTaskContainerBuilderClient := TaskContainerBuilder.NewTaskContainerBuilderServiceClient(conn)

		workflowTaskInfo := &TaskContainerBuilder.InputWorkflowTaskRequest {
			WorkflowId: val.WorkflowId,
			TaskNum: val.TaskNum,
			TaskName: val.TaskName,
			Image: val.Image,
			Cpu: val.Cpu,
			Mem: val.Mem,
			TaskOrder: val.TaskOrder,
			Env: val.Env,
			InputVector: val.InputVector,
			OutputVector: val.OutputVector,
			Args: val.Args,
            Labels: val.Labels,
            DurationTime: val.Duration,
            MinCpu: val.MinCpu,
            MinMem: val.MinMem,
		}
		//Access to this task request remotely
		podCreateResponse, err := visitTaskContainerBuilderClient.InputWorkflowTask(context.Background(), workflowTaskInfo)
		if podCreateResponse != nil {
			log.Printf("Return creating workflow status is %v\n",podCreateResponse.Result)
		} else {
			log.Println(err.Error())
		}
		//time.Sleep(200*time.Millisecond)
		_ = conn.Close()
	}
}
func visitTaskContainerBuilderRequest(wg *sync.WaitGroup, serverIp string, redisServer string, wf workflowTaskMap) {
	defer wg.Done()
	//1.Dial
	log.Println("Testing workflow task container...")
	wfSema <- 1
	//Send workflow following task's sequence.
	for i:= 0; i < len(wf); i++ {
		//for _, val := range wf {
		val := wf[uint64(i)]

		log.Printf("sending task: %v\n",val)
		//write Key-value data (TaskName:{startTime,duration,deadline,Cpu,Mem,InputVector,OutputVector})
		//writeDataInRedis(val,redisServer)
		conn, err := grpc.Dial(serverIp, grpc.WithInsecure())
		log.Println(serverIp)
		if err != nil {
			panic(err.Error())
		}
		visitTaskContainerBuilderClient := TaskContainerBuilder.NewTaskContainerBuilderServiceClient(conn)

		workflowTaskInfo := &TaskContainerBuilder.InputWorkflowTaskRequest {
			WorkflowId: val.WorkflowId,
			TaskNum: val.TaskNum,
			TaskName: val.TaskName,
			Image: val.Image,
			Cpu: val.Cpu,
			Mem: val.Mem,
			TaskOrder: val.TaskOrder,
			Env: val.Env,
			InputVector: val.InputVector,
			OutputVector: val.OutputVector,
			Args: val.Args,
			Labels: val.Labels,
			DurationTime: val.Duration,
			MinCpu: val.MinCpu,
			MinMem: val.MinMem,
		}
		podCreateResponse, err := visitTaskContainerBuilderClient.InputWorkflowTask(context.Background(), workflowTaskInfo)
		if podCreateResponse != nil {
			log.Printf("Return creating workflow status is %v\n",podCreateResponse.Result)
		} else {
			log.Println(err.Error())
		}
		//time.Sleep(200*time.Millisecond)
		_ = conn.Close()
	}
	<- 	wfSema
}

func writeDataInRedisFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from ", r)
	}
}

func writeDataInRedis(task WorkflowTask, redisAddress string){
	var uploadTaskData saveWorkflowTaskData
	var unMarshalData saveWorkflowTaskData
	defer writeDataInRedisFail()
	// build connection
	conn, err := redis.Dial("tcp", redisAddress)
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
		//This is the first task that has no parent node and is the entry task.
		start := uint64(time.Now().UnixNano()/1e6)
		uploadTaskData = saveWorkflowTaskData{
			StartTime: start,
			//the task-emulator container implements double 'task.Duration' via the stress tool.
			Duration: task.Duration*1000,
			Deadline: start + task.Duration*1000,
			AliveStatus: false,
			MilliCpu: task.Cpu,
			Memory: task.Mem,
		}
		log.Printf("task0-Data: %v\n",uploadTaskData)

	} else {
		//Find the largest deadline within 'task.InputVector', i.e., multiple ancestor tasks.
		tempTime := uint64(1)
		for _, inputIndex := range task.InputVector {
			log.Printf("task.InputVector: %v\n",task.InputVector)
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
		//The largest deadline among multiple ancestor tasks is used as its own start time.
		uploadTaskData = saveWorkflowTaskData{
			StartTime: tempTime,
			//the task-emulator container implements double 'task.Duration' via the stress tool.
			Duration: task.Duration*1000,
			Deadline: tempTime + task.Duration*1000,
			AliveStatus: false,
			MilliCpu: task.Cpu,
			Memory: task.Mem,
		}
	}
	//结构体序列化为字符串
	log.Println(uploadTaskData)
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

//build gRPC Server
func workflowInjectorServer(waiter *sync.WaitGroup) {
	defer waiter.Done()
	server := grpc.NewServer()
	log.Println("Build workflow task injector gRPC Server.")
	//Register the resource request service
	TaskContainerBuilder.RegisterWorkflowInjectorServiceServer(server,new(ResourceServiceImpl))
	//Listen on 7070 port
	lis, err := net.Listen("tcp", ":7070")
	log.Println("Listening local port 7070")
	if err != nil {
		panic(err.Error())
	}
	server.Serve(lis)
}
//read the json file of workflow definition from task container
func readDependencyMap() map[string]map[string][]string {
	var value4 []string
	//b, err := ioutil.ReadFile("D:\\GoProgram\\k8s-test\\dependency.json")
	b, err := ioutil.ReadFile("/config/dependency.json")
	if err != nil {
		panic(err)
	}
	whitelist := make(map[string]interface{})
	err = json.Unmarshal(b, &whitelist)
	if err != nil {
		panic(err)
	}
	dependencyMap := make(map[string]map[string][]string)
	dependency := make(map[string][]string)
	for key, value := range whitelist {
		val := value.(map[string]interface{})
		for key1, value2 := range val {
			v := value2.([]interface{})
			for _, value3 := range v {
				value4 = append(value4,value3.(string))
			}
			dependency[key1] = value4
			value4 = []string{}
		}
		dependencyMap[key] = dependency
		dependency = make(map[string][]string)
	}
	//for key, value := range dependencyMap {
	//	log.Println( key, ":", value)
	//}
	return dependencyMap
}
func main() {
	log.Println("workflow injecting module--->gRPC request--->task container builder")
	log.Println("Delay five seconds till the task container builder launches successfully.")
	time.Sleep(time.Second*5)

	wg := sync.WaitGroup{}
	waiter := sync.WaitGroup{}
	waiter.Add(1)
	//Start gRPC Server
	go workflowInjectorServer(&waiter) // Start Goroutine
    //get env of task number of this workflow
	taskNumber := os.Getenv("TASK_NUMBERS")
	log.Printf("taskNumber: %v\n",taskNumber)
	taskAccounts, err := strconv.Atoi(taskNumber)
	if err != nil {
		panic(err)
	}
	taskAccount := uint32(taskAccounts)
	/*Create workflow task map*/
	wfTaskMap = make(map[uint64]WorkflowTask)
	workflowMap = make(map[uint64]map[uint64]WorkflowTask)
	/*Obtain the map with DAG dependence from Configmap*/
	myDependencyMap := readDependencyMap()
    log.Println(myDependencyMap)
    /*The arg parameter is the number of workflows*/
	argNum := len(os.Args)
	if argNum > 2 {
		log.Println("Command args number is only one.")
	}
	argValue,err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	//allTasksRequestCpu, _ := strconv.ParseInt(os.Args[2],10,64)
	//allTasksRequestMem, _:= strconv.ParseInt(os.Args[3],10,64)
	//wg.Add(10)
	//set seed number,创建一个rand.Rand对象，避免多线程访问
	//numC := rand.New(rand.NewSource(time.Now().UnixNano()))
	//rand.Seed(time.Now().UnixNano())
	//taskCpuNum := uint64((numC.Intn(9)+5)*10)
	//taskMemNum := uint64((numC.Intn(9)+5)*10)
	//taskCpuNum := uint64(1200)
	//taskMemNum := uint64(1200)

	/*Define each workflow in turn*/
	for i := 0; i < argValue; i++ {
		//taskNum := uint32(numC.Intn(5)+5)
		taskNum := uint32(taskAccount)
		/*Define each task in turn and write the task map*/
		for j := 0; j < int(taskNum) ; j++ {
			cpuNum, err := strconv.Atoi(myDependencyMap[strconv.Itoa(j)]["cpuNum"][0])
			if err != nil {
				panic(err.Error())
			}
			memNum, err := strconv.Atoi(myDependencyMap[strconv.Itoa(j)]["memNum"][0])
			if err != nil {
				panic(err.Error())
			}
			minCpuValue, err := strconv.Atoi(myDependencyMap[strconv.Itoa(j)]["minCpu"][0])
			if err != nil {
				panic(err.Error())
			}
			minMemValue, err := strconv.Atoi(myDependencyMap[strconv.Itoa(j)]["minMem"][0])
			if err != nil {
				panic(err.Error())
			}
			//acquire the timeout(format:string) parameter of the task container
			durationTime := myDependencyMap[strconv.Itoa(j)]["timeout"][0]
			//append []string of args
		    operationArgs := append(myDependencyMap[strconv.Itoa(j)]["args"], "-t" , durationTime)
			////acquire the timeout(format:int) parameter of the task container
		    durationNum, err := strconv.Atoi(durationTime)
			if err != nil {
				panic(err.Error())
			}

			wfTaskMap[uint64(j)] = WorkflowTask{
				WorkflowId:"workflow-"+strconv.Itoa(i),
				TaskNum: taskNum,
				TaskName:"workflow-"+strconv.Itoa(i)+"-task-"+strconv.Itoa(j),
				//Image: "shanchenggang/task-emulator:latest",
				Image: myDependencyMap[strconv.Itoa(j)]["image"][0],
				Cpu: uint64(cpuNum),
				Mem: uint64(memNum),
				TaskOrder: uint32(j),
				Env: nil,
				InputVector: myDependencyMap[strconv.Itoa(j)]["input"],
				OutputVector: myDependencyMap[strconv.Itoa(j)]["output"],
				Args: operationArgs,
				Labels: map[string]string{myDependencyMap[strconv.Itoa(j)]["labelName"][0]:
					myDependencyMap[strconv.Itoa(j)]["labelValue"][0]},
				Duration: uint64(durationNum)*2,
				MinCpu: uint64(minCpuValue),
				MinMem: uint64(minMemValue),
			}
			//log.Println(workflowTaskMap[uint64(j)])
		}
		workflowMap[uint64(i)] = wfTaskMap
		wfTaskMap = make(map[uint64]WorkflowTask)
	//	for key, _ := range workflowTaskMap{
	//		delete(workflowTaskMap,key)
	//	}
	}
	//log.Println(workflowMap)
	for k, val := range workflowMap {
		log.Println("--------------------------------------")
		log.Printf("%v:%v\n",k,val)
	}
	/*Obtain Ip and port of taskContainerBuilder的IP*/
	TaskContainerBuilderServer := os.Getenv("TASK_CONTAINER_BUILDER_SERVICE_HOST")
	TaskContainerBuilderPort := os.Getenv("TASK_CONTAINER_BUILDER_SERVICE_PORT")
	taskContainerBuilderServerIp = TaskContainerBuilderServer + ":" + TaskContainerBuilderPort
	log.Println(taskContainerBuilderServerIp)
	//Acquire the env of Redis's Ip
	RedisServer := os.Getenv("REDIS_SERVER")
	RedisServerPort := os.Getenv("REDIS_PORT")
	redisIpPort := RedisServer + ":" + RedisServerPort
	log.Println(redisIpPort)
	//Herein, we can design the input pattern of workflowMap[i], such as constant arrival scenario,
	//linear arrival scenario and pyramid arrival scenario.
	//constant arrival scenario,
	batchNum := os.Getenv("BATCH_NUM")
	intervalTime := os.Getenv("INTERVAL_TIME")
	log.Printf("batchNum: %v\n",batchNum)
	log.Printf("intervalTime: %v\n",intervalTime)
	batchNumber, err := strconv.Atoi(batchNum)
	if err != nil {
		panic(err)
	}
	interval, err := strconv.Atoi(intervalTime)
	if err != nil {
		panic(err)
	}

//Constant arrival pattern
	begin := 0
	for i := 0; i < (argValue/batchNumber); i++ {
		wg.Add(batchNumber)
		for j := begin; j < batchNumber*(i+1); j++ {
			go visitTaskContainerBuilderRequest(&wg, taskContainerBuilderServerIp, redisIpPort, workflowMap[uint64(j)])

			if j>0 && (j+1) % batchNumber == 0 {
				begin = j + 1
				break
			}
		}
		wg.Wait()
		time.Sleep(time.Duration(interval)*time.Second)
	}
	waiter.Wait()
}
