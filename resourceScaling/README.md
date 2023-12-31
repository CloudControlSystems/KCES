
# Containerized workflow builder for Kubernetes
We have open-sourced the CWB, which can be incorporated into other workflow management systems for K8s as a docking module. 
We welcome you to download, learn, and work together to maintain CWB with us. If you use it for scientific research and 
engineering applications, please be sure to protect the copyright and indicate authors and source.

##Resource description

###./experiment

This directory includes the CWB deployment files. 
We incorporate all the deployment and log collection files into script files.
You only deploy and cleanup the CWB through 'deploy.sh' and 'clear.sh'. 
These script files automatically help you update the fields 'master.ip','name: system:node:' and 'server' included in
'resourceUsage.yaml.bak', 'rbac-deploy.yaml.bak', 'workflowInjector-Builder.yaml', 
and 'storageClass-nfs.yaml.bak'.

Firstly, you need to update the 'ipNode.txt' in line with your test cluster.
Here, our experimental environment consists of one Master and two nodes. 
In the K8s cluster, the node name is identified by the node's IP.

Master: 192.168.6.109, node1: 192.168.6.110, node2: 192.168.6.111

OS version: CentOS Linux release 7.8/Ubuntu 20.4, Kubernetes: v1.18.6/v1.19.6, Docker: 18.09.6.

####./experiment/cwb_test

The directory './deploy' includes the Yaml files corresponding to CWB, RBAC, resource usage rate, NFS, and workflow injection module.
We use the Configmap method in the Yaml file to inject workflow information (dependency.json) into the container of workflow injection module.

Refer to './deploy/task-dag7.yaml' for details.

steps:

##### a. update the K8s nodes' ip.

Update the 'ipNode.txt' in line with your test cluster.

##### b. ./deploy.sh

Deploy CWB into the K8s cluster. The './deploy/edit.sh' file firstly captures the Master's IP, 
updates the other corresponding files. Then it copies 'task-dag*.yaml.bak' to 'workflowInjector-Builder.yaml'.
The 'deploy.sh' file includes a series of 'Kubectl' commands.
During the workflow lifecycle, you can watch the execution states of workflow tasks. The following is the operation command.

'kubectl get pods -A --watch -o wide'

##### c. ./clear.sh

When the workflow is completed, you can run '.clear.sh' file to clean up the workflow information and obtain the log files. 

###./resourceUsage

This directory includes the source codes of the resource gathering module.
You can build the Docker image by the 'Dockerfile' file or pull the image of this module from Docker Hub.

'docker pull shanchenggang/resource-usage:v1.0'

###./TaskContainerBuilder

This directory includes the source codes of CWB.
You can build the Docker image by the 'Dockerfile' file or pull the image of this module from Docker Hub.

'docker pull shanchenggang/task-container-builder:v5.0'

###./usage_deploy

This directory file includes the deployment file of the resource gather module.
Note that the './experiment' directory has included this deployment file for three submission approaches.

###./WorkflowInjector

This directory includes the image address of the workflow injection module.
You can build the Docker image by the 'Dockerfile' file or pull the image of this module from Docker Hub.

'docker pull shanchenggang/workflow-injector:v5.0'
