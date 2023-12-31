shanchenggang/task-emulator:v1.0或者shanchenggang/task-emulator-arm64:v1.0 跨云边的 stress计算任务

shanchenggang/task-emulator:v2.0  带环路的任务，gRPC service访问其他节点
shanchenggang/task-emulator:v3.0  带环路的任务，gRPC service访问其他节点,service环境参数采用编程注入的方式

(shanchenggang/workflow-injector:v7.0 不带label的工作流任务，注入)
shanchenggang/workflow-injector:v7.2 单个工作流的依次注入，延迟5s,启动gRPC, 带label的工作流任务(通过label调度到指定yun'he的云节点或边节点)

shanchenggang/task-container-builder:v7.1 能处理连续的跨云边的工作流容器化

shanchenggang/task-container-builder:v7.2 能处理NFS存储共享的连续的工作流云边的容器化
shanchenggang/task-container-builder:v7.3 单个工作流容器化，适用于云带环路的工作流，工作流任务一次注入启动。
shanchenggang/task-container-builder:v7.4 单个工作流容器化，适用于云和云边的带环路的工作流，工作流任务一次注入启动。每个工作流任务对应的Service参数编程注入对应的工作流任务pod。

shanchenggang/service-builder:v1.0  提前在指定的namespace下生成与工作流任务相等数量的service，CWB生成的工作流Pod会通过label与已经存下的service绑定。task-container-builder pod能够获取service环境
变量。

shanchenggang/task-container-builder:baseline 云边工作流引擎资源分配不采用资源伸缩算法
shanchenggang/task-container-builder:scaling 云边工作流引擎资源分配采用资源伸缩算法
