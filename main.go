/*
Copyright 2016 Juniper

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

*/
package main

import (
        "time"
	"fmt"
	"net"
	"strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
        "k8s.io/api/core/v1"
        "gopkg.in/yaml.v2"
)


func main(){
   err := createConfig()
   if err != nil {
     panic(err.Error())
   }
}

func createConfig() error{
        return retry(10, time.Second, func() error {
	  config, err := rest.InClusterConfig()
	  if err != nil {
		panic(err.Error())
	  }
	  clientset, err := kubernetes.NewForConfig(config)
	  if err != nil {
		panic(err.Error())
	  }
          nodeList, err := clientset.CoreV1().Nodes().List(
                    metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/master=",})
          if err != nil {
            return err
          }
          var masterNodes []string
          for _, element := range nodeList.Items {
                  masterNodes = append(masterNodes, element.Name)
          }

          kubeadmConfigMapClient := clientset.CoreV1().ConfigMaps("kube-system")
          kcm, err := kubeadmConfigMapClient.Get("kubeadm-config", metav1.GetOptions{})
          clusterConfig := kcm.Data["ClusterConfiguration"]
          clusterConfigByte := []byte(clusterConfig)
          clusterConfigMap := make(map[interface{}]interface{})
          err = yaml.Unmarshal(clusterConfigByte, &clusterConfigMap)
          if err != nil {
            return err
          }
          controlPlaneEndpoint := clusterConfigMap["controlPlaneEndpoint"].(string)
          controlPlaneEndpointHost, controlPlaneEndpointPort, _ := net.SplitHostPort(controlPlaneEndpoint)
          clusterName := clusterConfigMap["clusterName"].(string)

          networkConfig := make(map[interface{}]interface{})
          networkConfig = clusterConfigMap["networking"].(map[interface{}]interface{})
          podSubnet := networkConfig["podSubnet"].(string)
          serviceSubnet := networkConfig["serviceSubnet"].(string)

          configMap := &v1.ConfigMap{
              ObjectMeta: metav1.ObjectMeta{
                  Name: "contrailcontrollernodes",
                  Namespace: "contrail",
              },
              Data: map[string]string{
                  "CONTROLLER_NODES": strings.Join(masterNodes,","),
                  "KUBERNETES_API_SERVER": controlPlaneEndpointHost,
                  "KUBERNETES_API_SECURE_PORT": controlPlaneEndpointPort,
                  "KUBERNETES_POD_SUBNETS": podSubnet,
                  "KUBERNETES_SERVICE_SUBNETS": serviceSubnet,
                  "KUBERNETES_CLUSTER_NAME": clusterName,
              },
          }

          configMapClient := clientset.CoreV1().ConfigMaps("contrail")
          cm, err := configMapClient.Get("contrailcontrollernodes", metav1.GetOptions{})
          if err != nil {
            configMapClient.Create(configMap)
            fmt.Printf("Created %s\n", cm.Name)
          } else {
            configMapClient.Update(configMap)
            fmt.Printf("Updated %s\n", cm.Name)
          }
          return nil
        })
}
