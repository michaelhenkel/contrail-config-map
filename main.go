/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
        "time"
	"fmt"
	"net"
        "net/url"
	"strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
        "k8s.io/api/core/v1"
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
          u, err := url.Parse(config.Host)
          host, port, _ := net.SplitHostPort(u.Host)
          fmt.Printf("Host: %s, Port: %s\n", host, port)
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
          fmt.Println(strings.Join(masterNodes,","))
          configMap := &v1.ConfigMap{
              ObjectMeta: metav1.ObjectMeta{
                  Name: "contrailcontrollernodes",
                  Namespace: "contrail",
              },
              Data: map[string]string{
                  "CONTROLLER_NODES": strings.Join(masterNodes,","),
                  "KUBERNETES_API_SERVER": host,
                  "KUBERNETES_API_SECURE_PORT": port,
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
