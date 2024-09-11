#!/bin/bash

function dev() {
    make generate

    make manifests

    make uninstall

    make install

    make run
}

function build() {
    
    make docker-build docker-push IMG=registry.cn-chengdu.aliyuncs.com/greatsql/greatsql-operator:$version
    
    make deploy IMG=registry.cn-chengdu.aliyuncs.com/greatsql/greatsql-operator:$version
}