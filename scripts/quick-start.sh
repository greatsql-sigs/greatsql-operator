#!/bin/bash

namespace="greatsql"
crdDir="config/crd/bases"
operatorFile="config/manager/manager.yaml"

set -e

function info() {
  echo -e "\033[32m$1\033[0m"
}

function error() {
  echo -e "\033[31m$1\033[0m"
}

function isNamespaceExist() {
  kubectl get namespace $namespace > /dev/null 2>&1
  if [ $? -ne 0 ]; then
    info "create namespace $namespace"
    kubectl create namespace $namespace
  fi
}

function applyCRD() {
  local crdFile=$1
  kubectl get crd $(basename $crdFile .yaml) > /dev/null 2>&1
  if [ $? -ne 0 ]; then
    info "Applying CRD from $crdFile"
    kubectl apply -f $crdFile
  else
    info "CRD already exists: $(basename $crdFile .yaml)"
  fi
}

function isOperatorManagerExist() {
  kubectl get deployment -n $namespace greatsql-operator > /dev/null 2>&1
  if [ $? -ne 0 ]; then
    info "create operator manager"
    kubectl apply -f $operatorFile
  fi
}

function installOperator() {
    echo "Select GreatSQL mode:"
    echo "1) Single Instance"
    echo "2) Replication Cluster"
    echo "3) MySQL Group Replication Cluster"

    read -p "Enter your choice (1/2/3): " choice

    case $choice in
        1)
            info "Install GreatSQL Single Instance"
            applyCRD $crdDir/greatsql.greatsql.cn_singleinstances.yaml
            ;;
        2)
            info "Install GreatSQL Replication Cluster"
            applyCRD $crdDir/greatsql_v1alpha1_greatsql_cr_replication_cluster.yaml
            ;;
        3)
            info "Install GreatSQL MySQL Group Replication Cluster"
            applyCRD $crdDir/greatsql.greatsql.cn_groupreplicationclusters.yaml
            ;;
        *)
            error "Invalid choice"
            exit 1
            ;;
    esac

    isNamespaceExist
    applyCRD $crdFile
    isOperatorManagerExist
}

function inspectManagerStatus() {
  info "inspect operator manager status"
  kubectl get deployment -n $namespace greatsql-operator
  kubectl get pod -n $namespace -l control-plane=greatsql-operator
}

installOperator
inspectManagerStatus