#!/bin/bash
#get master ip in this K8s cluster
iface=`ip a | grep "state UP" | awk '{print $2}'| awk -F: '{print $1}' | awk 'NR==1{print}'`
ip_master=`ip a | grep -E "inet *.+$iface" | grep -w inet | awk '{print $2}' | awk -F '/' '{print $1}'`

cp serviceBuilder.yaml.bak serviceBuilder.yaml
cp rbac-deploy.yaml.bak rbac-deploy.yaml
cp storageClass-nfs.yaml.bak storageClass-nfs.yaml
cp resourceUsage.yaml.bak resourceUsage.yaml
cp Montage.yaml.bak workflowInjector-Builder.yaml
#cp Epigenomics.yaml.bak workflowInjector-Builder.yaml
#cp CyberShake.yaml.bak workflowInjector-Builder.yaml
#cp LIGO.yaml.bak workflowInjector-Builder.yaml
#cp task-dag7.yaml.bak workflowInjector-Builder.yaml
#cp rasp-temperature.yaml.bak workflowInjector-Builder.yaml
#cp science-computing.yaml.bak workflowInjector-Builder.yaml
#cp cloudcontrol-test.yaml.bak workflowInjector-Builder.yaml
#cp nfs-task.yaml.bak workflowInjector-Builder.yaml
#替换IP

sed -i "s/0.0.0.0/$ip_master/g" storageClass-nfs.yaml
#sed -i "s/0.0.0.0/$ip_master/g" rabc-deploy.yaml
sed -i "s/0.0.0.0/$ip_master/g" resourceUsage.yaml
#sed -i "s/0.0.0.0/$ip_master/g" workflowInjector-Builder.yaml
sed -i "s/0.0.0.0/$ip_master/g" workflowInjector-Builder.yaml


ip1=`echo $ip_master | awk -F . '{print $1}'`
ip2=`echo $ip_master | awk -F . '{print $2}'`
ip3=`echo $ip_master | awk -F . '{print $3}'`
ip4=`echo $ip_master | awk -F . '{print $4}'`


for((i=0;i<=6;i++))
do
  ip0=$(($ip4+$i))
  if [ $ip0 -lt 256 ]
  then
    ip10=$ip1.$ip2.$ip3.$ip0
    sed -i "s/$i.$i.$i.$i/$ip10/g" rbac-deploy.yaml
  else
    ip0=$(($ip0-256))
    ip33=$(($ip3+1))
    ip10=$ip1.$ip2.$ip33.$ip0
    sed -i "s/$i.$i.$i.$i/$ip10/g" rbac-deploy.yaml
  fi
done

