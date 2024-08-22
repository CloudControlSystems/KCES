package k8sResource

import (
	"fmt"
	_ "golang.org/x/net/context"
	_"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"

	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"

)
//testing k8s cluster API by client-go packages
var clientset *kubernetes.Clientset
type resourceRequest struct {
	MilliCpu uint64
	Memory uint64
	EphemeralStorage uint64
}
type NodeResidualResource struct {
	MilliCpu uint64
	Memory uint64
}
type NodeAllocateResource struct {
	MilliCpu uint64
	Memory uint64
}
type NodeUsedResource struct {
	MilliCpu uint64
	Memory uint64
}
var resourceRequestNum resourceRequest
var resourceAllocatableNum resourceRequest
type ResidualResourceMap map[string]NodeResidualResource

type NodeUsedResourceMap map[string]NodeUsedResource
type NodeAllocateResourceMap map[string]NodeAllocateResource
var readK8sConfigNum uint64
//GetK8sApiResource


func getK8sClient(configfile string) *kubernetes.Clientset {
	//k8sconfig= flag.String("k8sconfig","/opt/kubernetes/cfg/kubelet.kubeconfig","kubernetes config file path")
	//flag.Parse()
	//var k8sconfig string
	//if configfile == "/kubelet.kubeconfig" {
		//k8sconfig, err  := filepath.Abs(filepath.Dir("/opt/kubernetes/cfg/kubelet.kubeconfig"))

	k8sconfig, err  := filepath.Abs(filepath.Dir("/opt/kubernetes/cfg/kubelet.kubeconfig"))
		if err != nil {
			panic(err.Error())
		}
		config, err := clientcmd.BuildConfigFromFlags("",k8sconfig + configfile)
		if err != nil {
			log.Println(err)
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalln(err)
		} else {
			log.Println(configfile)
			log.Println("Connect this cluster's k8s successfully.")
		}
		return clientset

	//viper.AddConfigPath("/opt/kubernetes/cfg/")     //设置读取的文件路径
	//viper.SetConfigName("kubelet") //设置读取的文件名
	//viper.SetConfigType("yaml")   //设置文件的类型
	//k8sconfig := viper.ReadInConfig()
	//viper.WatchConfig()
}
func GetInformerK8sClient(configfile string) *kubernetes.Clientset {
	//k8sconfig= flag.String("k8sconfig","/opt/kubernetes/cfg/kubelet.kubeconfig","kubernetes config file path")
	//flag.Parse()
	//var k8sconfig string
	//if configfile == "/kubelet.kubeconfig" {
	//k8sconfig, err  := filepath.Abs(filepath.Dir("/opt/kubernetes/cfg/kubelet.kubeconfig"))

	k8sconfig, err  := filepath.Abs(filepath.Dir("/etc/kubernetes/kubelet.kubeconfig"))
	if err != nil {
		panic(err.Error())
	}
	config, err := clientcmd.BuildConfigFromFlags("",k8sconfig + configfile)
	if err != nil {
		log.Println(err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println(configfile)
		log.Println("Connect this cluster's k8s successfully.")
	}
	return clientset

	//viper.AddConfigPath("/opt/kubernetes/cfg/")     //设置读取的文件路径
	//viper.SetConfigName("kubelet") //设置读取的文件名
	//viper.SetConfigType("yaml")   //设置文件的类型
	//k8sconfig := viper.ReadInConfig()
	//viper.WatchConfig()
}
func GetRemoteK8sClient() *kubernetes.Clientset {
	//k8sconfig= flag.String("k8sconfig","/opt/kubernetes/cfg/kubelet.kubeconfig","kubernetes config file path")
	//flag.Parse()
	//var k8sconfig string
	k8sconfig, err  := filepath.Abs(filepath.Dir("/etc/kubernetes/kubelet.kubeconfig"))
	if err != nil {
		panic(err.Error())
	}
	config, err := clientcmd.BuildConfigFromFlags("",k8sconfig+ "/kubelet.kubeconfig")
	if err != nil {
		log.Println(err)
	}
	//viper.AddConfigPath("/opt/kubernetes/cfg/")     //设置读取的文件路径
	//viper.SetConfigName("kubelet") //设置读取的文件名
	//viper.SetConfigType("yaml")   //设置文件的类型
	//k8sconfig := viper.ReadInConfig()
	//viper.WatchConfig()
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println("Connect k8s success.")
	}
	return clientset
}

func recoverPodListerFail() {
	if r := recover(); r!= nil {
		log.Println("recovered from ", r)
	}
}
func GetK8sEachNodeResource(podLister v1.PodLister,nodeLister v1.NodeLister,
	ResourceMap ResidualResourceMap) ResidualResourceMap {
	defer recoverPodListerFail()

	//从podlister,nodelister中获取所有items
	podList, err := podLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}
	nodeList, err := nodeLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}
    for key, val := range ResourceMap {
    	currentNodePodsResourceSum := getEachNodePodsResourceRequest(podList, key)
    	currentNodeAllocatableResource := getEachNodeAllocatableNum(nodeList, key)
    	val.MilliCpu = currentNodeAllocatableResource.MilliCpu - currentNodePodsResourceSum.MilliCpu
    	val.Memory = currentNodeAllocatableResource.Memory - currentNodePodsResourceSum.Memory
    	ResourceMap[key] = NodeResidualResource{val.MilliCpu, val.Memory/1024/1024}
	}
    //log.Println(ResourceMap)
	return ResourceMap
}

func GetEachNodeResource(podLister v1.PodLister,nodeLister v1.NodeLister,
	NodeUsedResourceMap NodeUsedResourceMap,
	NodeAllocateResourceMap NodeAllocateResourceMap)(NodeUsedResourceMap,NodeAllocateResourceMap) {
	defer recoverPodListerFail()

	//从podlister,nodelister中获取所有items
	podList, err := podLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}
	nodeList, err := nodeLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}

	for key, val := range NodeUsedResourceMap {
		currentNodePodsResourceSum := getEachNodePodsResourceRequest(podList, key)
		val.MilliCpu =  currentNodePodsResourceSum.MilliCpu
		val.Memory = currentNodePodsResourceSum.Memory
		NodeUsedResourceMap[key] = NodeUsedResource{val.MilliCpu, val.Memory/1024/1024}
	//log.Println(NodeUsedResourceMap)
	}

	for key, val1 := range NodeAllocateResourceMap {
		currentNodeAllocatableResource := getEachNodeAllocatableNum(nodeList, key)
		val1.MilliCpu = currentNodeAllocatableResource.MilliCpu
		val1.Memory = currentNodeAllocatableResource.Memory
		NodeAllocateResourceMap[key] = NodeAllocateResource{val1.MilliCpu, val1.Memory/1024/1024}
	//log.Println(NodeAllocateResourceMap)
	}
	//log.Println(NodeUsedResourceMap)
	//log.Println(NodeAllocateResourceMap)
	return NodeUsedResourceMap, NodeAllocateResourceMap
}

func GetK8sApiResource(podLister v1.PodLister,nodeLister v1.NodeLister) resourceRequest {
	//golang异常处理机制，利用recover捕获panic
	defer recoverPodListerFail()

	//从podlister,nodelister中获取所ap)
	//	return NodeUsedResourceMap, NodeAllocateR有items
	podList, err := podLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}
	nodeList, err := nodeLister.List(labels.Everything())
	if err != nil {
		log.Println(err)
		panic(err.Error())
	}

	//get nodes info in whole master cluster
	//nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	//if err != nil {
	//	log.Println(err.Error())
	//}
	// print nodes info in whole master cluster
	//printK8SNodeInfo(*nodes)
	//fmt.Println(pods.Items[0].Name)
	//fmt.Println(pods.Items[0].CreationTimestamp)
	//fmt.Println(pods.Items[0].Labels)
	//fmt.Println(pods.Items[0].Namespace)
	//fmt.Println(pods.Items[0].Status.PodIP)
	//fmt.Println(pods.Items[0].Status.StartTime)
	//fmt.Println(pods.Items[0].Status.Phase)
	//fmt.Println(pods.Items[0].Status.ContainerStatuses[0].RestartCount)   //重启次数
	//fmt.Println(pods.Items[0].Status.ContainerStatuses[0].Image) //获取重启时间
	//fmt.Println(pods.Items[0].Status.ContainerStatuses[0].Size())
	//get nodes info
	//for _, node := range nodes.Items {
	//	//fmt.Printf("Name: %s, Status: %s, CreateTime: %s\n", node.ObjectMeta.Name, node.Status.Phase, node.ObjectMeta.CreationTimestamp)
	//
	//}
	//fmt.Println(nodes.Items[1].Name)
	//fmt.Println(nodes.Items[1].CreationTimestamp)    //加入集群时间
	//fmt.Println(nodes.Items[1].Status.NodeInfo)
	//fmt.Println(nodes.Items[1].Status.Conditions[len(nodes.Items[0].Status.Conditions)-1].Type)
	//fmt.Println("*********************************")
	//fmt.Println(nodes.Items[1].Status.Allocatable.Cpu().String())
	//fmt.Println(nodes.Items[1].Status.Allocatable.Memory().String())
	//fmt.Println(nodes.Items[1].Status.Capacity.Cpu().String())
	//fmt.Println(nodes.Items[1].Status.Capacity.Memory().String())
	//fmt.Println("------------------------------------------------------------------")
	allPodsResourceRequest := getPodsResourceRequest(podList)
	//log.Printf("allPodsResourceRequest cpuNum =%dm,allPodsResourceRequest memNum =%dMi\n",
	//	allPodsResourceRequest.MilliCpu,allPodsResourceRequest.Memory/1024/1024)

	//Node1PodsResourceRequest := getNode1PodsResourceRequest(podList)
	//log.Printf("Node1PodsResourceRequest cpuNum =%dm,Node1PodsResourceRequest memNum =%dMi\n",
	//	Node1PodsResourceRequest.MilliCpu,Node1PodsResourceRequest.Memory/1024/1024)
	//Node2PodsResourceRequest := getNode2PodsResourceRequest(podList)
	//log.Printf("Node2PodsResourceRequest cpuNum =%dm,Node2PodsResourceRequest memNum =%dMi\n",
	//	Node2PodsResourceRequest.MilliCpu,Node2PodsResourceRequest.Memory/1024/1024)
	allNodesResourceAllocatable := getNodesAllocatableNum(nodeList)
	//log.Printf("allNodesResourceAllocatable cpuNum =%dm,allNodesResourceAllocatable memNum =%dMi\n",
	//	allNodesResourceAllocatable.MilliCpu,allNodesResourceAllocatable.Memory/1024/1024)
	//Node1ResourceAllocatable := getNode1AllocatableNum(nodeList)
	//log.Printf("Node1ResourceAllocatable cpuNum =%dm,Node1ResourceAllocatable memNum =%dMi\n",
	//	Node1ResourceAllocatable.MilliCpu,Node1ResourceAllocatable.Memory/1024/1024)
	//Node2ResourceAllocatable := getNode2AllocatableNum(nodeList)
	//log.Printf("Node2ResourceAllocatable cpuNum =%dm,Node2ResourceAllocatable memNum =%dMi\n",
	//	Node2ResourceAllocatable.MilliCpu,Node2ResourceAllocatable.Memory/1024/1024)

	//剩余资源
	//fmt.Printf("Residual Cpu = %dm\n", allNodesResourceAllocatable.MilliCpu-allPodsResourceRequest.MilliCpu)
	//fmt.Printf("Residual Memory = %dMi\n", (allNodesResourceAllocatable.Memory-allPodsResourceRequest.Memory)/1024/1024)
	readK8sConfigNum++
	log.Println("--------------------------------------------------------")
	log.Printf("get k8s resource in %dth time through informer.\n",readK8sConfigNum)
	return resourceRequest{
		MilliCpu: allNodesResourceAllocatable.MilliCpu - allPodsResourceRequest.MilliCpu,
		Memory:                     (allNodesResourceAllocatable.Memory - allPodsResourceRequest.Memory) / 1024 / 1024,
		EphemeralStorage:           allNodesResourceAllocatable.EphemeralStorage - allPodsResourceRequest.EphemeralStorage,

		//Node1ResidualCpuPercentage:(Node1ResourceAllocatable.MilliCpu-Node1PodsResourceRequest.MilliCpu)*100/Node1ResourceAllocatable.MilliCpu,
		//Node1ResidualMemPercentage:(Node1ResourceAllocatable.Memory-Node1PodsResourceRequest.Memory)*100/Node1ResourceAllocatable.Memory,
		//Node2ResidualCpuPercentage:(Node2ResourceAllocatable.MilliCpu-Node2PodsResourceRequest.MilliCpu)*100/Node2ResourceAllocatable.MilliCpu,
		//Node2ResidualMemPercentage:(Node2ResourceAllocatable.Memory-Node2PodsResourceRequest.Memory)*100/Node2ResourceAllocatable.Memory}
	}

}
func printK8SPodInfo(pods corev1.PodList) {
	fmt.Println("--------------------------------------------------------------------------------")

	const format = "%v\t%v\t%v\t%v\t%v\t%v\t\n"
	tw := new(tabwriter.Writer).Init(os.Stdout,0,8,2,' ',0)
	fmt.Fprintf(tw,format,"Namespace","Name","Status","Age","PodIP","Node")
	fmt.Fprintf(tw,format,"------","------","-----","----","-------","-----")
	for _, r := range pods.Items {
		fmt.Fprintf(tw,format,r.Namespace,r.Name,r.Status.Phase,r.Status.StartTime,r.Status.PodIP, r.Spec.NodeName)
	}
	tw.Flush()

}
func printK8SNodeInfo(nodes corev1.NodeList) {
	fmt.Println("--------------------------------------------------------------------------------")
	const format1 = "%v\t%v\t%v\t%v\t%v\t%v\t%v\t\n"
	tw1 := new(tabwriter.Writer).Init(os.Stdout,0,8,2,' ',0)
	fmt.Fprintf(tw1,format1,"Name","Status","CreateTime",
		"Allocatable.Cpu","Allocatable.Memory","Capacity.Cpu","Capacity.Memory")
	fmt.Fprintf(tw1,format1,"------","------","---------","----","----","----","----")
	for _, r := range nodes.Items {
		fmt.Fprintf(tw1,format1,r.Name,r.Status.Conditions[len(r.Status.Conditions)-1].Type,
			r.CreationTimestamp,r.Status.Allocatable.Cpu(),
			r.Status.Allocatable.Memory(),r.Status.Capacity.Cpu(),
			r.Status.Capacity.Memory())
	}
	tw1.Flush()
}
//遍历集群所有pods，获取集群所有pods的资源request值
func getPodsResourceRequest(pods []*corev1.Pod) resourceRequest {

	defer recoverPodListerFail()
	resourceRequestNum := resourceRequest{0, 0, 0}
	for _, pod := range pods {
		if pod.Status.HostIP != "192.168.6.109" {
		//if (pod.Status.HostIP != "121.250.173.190")&&(pod.Status.HostIP != "121.250.173.191")&&(pod.Status.HostIP != "121.250.173.192"){
			if (pod.Status.Phase == "Running")||(pod.Status.Phase == "Pending"){
				for _, container := range pod.Spec.Containers {
					resourceRequestNum.MilliCpu += uint64(container.Resources.Requests.Cpu().MilliValue())
					resourceRequestNum.Memory += uint64(container.Resources.Requests.Memory().Value())
					resourceRequestNum.EphemeralStorage += uint64(container.Resources.Requests.StorageEphemeral().Value())
				}
				for _, initContainer := range pod.Spec.InitContainers {
					resourceRequestNum.MilliCpu += uint64(initContainer.Resources.Requests.Cpu().MilliValue())
					resourceRequestNum.Memory += uint64(initContainer.Resources.Requests.Memory().Value())
				}
				//log.Printf("cpu = %d\n",resourceRequestNum.MilliCpu)
				//log.Printf("mem = %d\n",resourceRequestNum.Memory/1024/1024)
			}
		}
	}
	return resourceRequestNum
}
//遍历集群Node1节点所有pods，获取集群所有pods的资源request值
func getEachNodePodsResourceRequest(pods []*corev1.Pod, nodeName string) resourceRequest {
	defer recoverPodListerFail()
	resourceRequestNum = resourceRequest{0, 0, 0}
	for _, pod := range pods {
		if pod.Status.HostIP == nodeName {
			if (pod.Status.Phase == "Running")||(pod.Status.Phase == "Pending"){
				for _, container := range pod.Spec.Containers {
					resourceRequestNum.MilliCpu += uint64(container.Resources.Requests.Cpu().MilliValue())
					resourceRequestNum.Memory += uint64(container.Resources.Requests.Memory().Value())
					resourceRequestNum.EphemeralStorage += uint64(container.Resources.Requests.StorageEphemeral().Value())
				}
				for _, initContainer := range pod.Spec.InitContainers {
					resourceRequestNum.MilliCpu += uint64(initContainer.Resources.Requests.Cpu().MilliValue())
					resourceRequestNum.Memory += uint64(initContainer.Resources.Requests.Memory().Value())
				}
				//fmt.Printf("cpu = %d\n",resourceRequestNum.MilliCpu)
				//fmt.Printf("mem = %d\n",resourceRequestNum.Memory/1024/1024)
			}
		}
		//log.Printf("This %s's HostIP is %s .\n",pod.Name,pod.Status.HostIP)
	}
	return resourceRequestNum
}

//遍历集群Node2节点所有pods，获取集群所有pods的资源request值
func getNode2PodsResourceRequest(pods []*corev1.Pod) resourceRequest {
	resourceRequestNum := resourceRequest{0, 0, 0}
	for _, pod := range pods {
		if pod.Status.HostIP == "121.250.173.195"{
			if pod.Status.Phase == "Running" {
				for _, container := range pod.Spec.Containers {
					resourceRequestNum.MilliCpu += uint64(container.Resources.Requests.Cpu().MilliValue())
					resourceRequestNum.Memory += uint64(container.Resources.Requests.Memory().Value())
					resourceRequestNum.EphemeralStorage += uint64(container.Resources.Requests.StorageEphemeral().Value())
				}
				for _, initContainer := range pod.Spec.InitContainers {
					resourceRequestNum.MilliCpu += uint64(initContainer.Resources.Requests.Cpu().MilliValue())
					resourceRequestNum.Memory += uint64(initContainer.Resources.Requests.Memory().Value())
				}
				//fmt.Printf("cpu = %d\n",resourceRequestNum.MilliCpu)
				//fmt.Printf("mem = %d\n",resourceRequestNum.Memory/1024/1024)
			}
		}
		//log.Printf("This %s's HostIP is %s .\n",pod.Name,pod.Status.HostIP)
	}
	return resourceRequestNum
}
//获取集群所有nodes的allocatable资源值
func getNodesAllocatableNum(nodes []*corev1.Node) resourceRequest {
	resourceAllocatableNum = resourceRequest{0,0,0}
	for _, nod := range nodes {
        //if nod.Name[0:9] != "admiralty" {
        //if nod.Name[0:10] == "k8s-2-node" {
		if nod.Name[0:9] == "k8s4-node" {
			//if (nod.Name != "121.250.173.190")&&(nod.Name != "121.250.173.191")&&(nod.Name != "121.250.173.192"){
			resourceAllocatableNum.MilliCpu += uint64(nod.Status.Allocatable.Cpu().MilliValue())
			resourceAllocatableNum.Memory += uint64(nod.Status.Allocatable.Memory().Value())
			resourceAllocatableNum.EphemeralStorage += uint64(nod.Status.Allocatable.StorageEphemeral().Value())
			//log.Printf("Node-%s: Allocatable CpuNum = %dm,Allocatable MemNum = %dMi\n",
			//nod.Name, nod.Status.Allocatable.Cpu().MilliValue(),nod.Status.Allocatable.Memory().Value()/1024/1024)

		}
	}
	//log.Printf("Node1 and Node2: Allocatable CpuNum = %dm,Allocatable MemNum = %dMi\n",
	//	resourceAllocatableNum.MilliCpu,resourceAllocatableNum.Memory/1024/1024)
	return resourceAllocatableNum
}
//获取集群node的allocatable资源值
func getEachNodeAllocatableNum( nodes []*corev1.Node, nodeName string) resourceRequest {
	defer recoverPodListerFail()
	resourceAllocatableNum = resourceRequest{0,0,0}
	for _, nod := range nodes {
		//if nod.Name[0:9] != "admiralty" {
		//if nod.Name[0:12] == "k8s-2-node-1" {
		if nod.Name == nodeName {
		//if nod.Name == "k8s4-node1" && nodeName == "192.168.6.110" {
				resourceAllocatableNum.MilliCpu = uint64(nod.Status.Allocatable.Cpu().MilliValue())
				resourceAllocatableNum.Memory = uint64(nod.Status.Allocatable.Memory().Value())
				resourceAllocatableNum.EphemeralStorage = uint64(nod.Status.Allocatable.StorageEphemeral().Value())
		//}else {
		//	if nod.Name == "k8s4-node2" && nodeName == "192.168.6.111" {
		//		resourceAllocatableNum.MilliCpu = uint64(nod.Status.Allocatable.Cpu().MilliValue())
		//		resourceAllocatableNum.Memory = uint64(nod.Status.Allocatable.Memory().Value())
		//		resourceAllocatableNum.EphemeralStorage = uint64(nod.Status.Allocatable.StorageEphemeral().Value())
		//	}
	    }
		//log.Printf("This is %s.\n", nod.Name)
		//log.Printf("%s： Allocatable CpuNum = %dm,Allocatable MemNum = %dMi\n",
		//	nod.Name, nod.Status.Allocatable.Cpu().MilliValue(),nod.Status.Allocatable.Memory().Value()/1024/1024)
	}
	return resourceAllocatableNum
}
//获取集群所有node2的allocatable资源值
func getNode2AllocatableNum(nodes []*corev1.Node) resourceRequest {

	resourceAllocatableNum := resourceRequest{0,0,0}
	for _, nod := range nodes {
		//if nod.Name[0:9] != "admiralty" {
		//if nod.Name[0:12] == "k8s-2-node-2" {
		if nod.Name== "121.250.173.195" {
			resourceAllocatableNum.MilliCpu = uint64(nod.Status.Allocatable.Cpu().MilliValue())
			resourceAllocatableNum.Memory = uint64(nod.Status.Allocatable.Memory().Value())
			resourceAllocatableNum.EphemeralStorage = uint64(nod.Status.Allocatable.StorageEphemeral().Value())
		}
		//log.Printf("This is %s.\n", nod.Name)
		//log.Printf("Current node2: %s,Allocatable CpuNum = %dm,Allocatable MemNum = %dMi\n",
		//	nod.Name, nod.Status.Allocatable.Cpu().MilliValue(),nod.Status.Allocatable.Memory().Value()/1024/1024)
	}
	return resourceAllocatableNum
}

