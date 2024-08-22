package taskResponse

import "google.golang.org/protobuf/runtime/protoimpl"

type taskResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	//成功创建 pod 的状态码，1表示成功
	Result uint32 `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	//pod共享存储路径
	VolumePath string `protobuf:"bytes,2,opt,name=volumePath,proto3" json:"volumePath,omitempty"`
	//在失败状态下，可以设置状态码
	//成功状态（result>=1），客户端不关系此字段，置为 0 即可
	ErrNo uint32 `protobuf:"varint,3,opt,name=err_no,json=errNo,proto3" json:"err_no,omitempty"`
}
//定义工作流任务结构体
type workflowTask struct {
	//workflow的ID
	WorkflowId string
	//taskNum
	TaskNum uint32
	//taskName
	TaskName string
	Image string
	//任务镜像
	//基本单位 millicore(1Core=1000millicore)
	Cpu uint64
	//基本单位 MiB
	Mem uint64
	//任务在工作流的执行顺序
	TaskOrder uint32
	//需要输入给 POD 的环境变量
	Env map[string]string
	// 输入向量
	InputVector []string
	// 输出向量
	OutputVector []string
	// 任务pod所需参数
	Args []string
	//pod label
	Labels map[string]string
	//duration (second)
	Duration uint64
}
type  WorkflowTaskMap map[uint32] workflowTask
var workflowTaskMap = make(map[uint32] workflowTask)

func main() {
	
}
