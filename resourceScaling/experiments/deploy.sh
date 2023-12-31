#!/bin/bash
cd ./deploy && ./edit.sh
#deploy
cd ..
kubectl apply -f ./deploy/rbac-deploy.yaml
kubectl apply -f ./deploy/storageClass-nfs.yaml
kubectl apply -f ./deploy/serviceBuilder.yaml
kubectl apply -f ./deploy/resourceUsage.yaml
sleep 5s
kubectl apply -f ./deploy/workflowInjector-Builder.yaml

