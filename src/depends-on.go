package main

import (
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

func main() {

	services := os.Args[1:]
	nsb, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		panic(err.Error())
	}
	ns := string(nsb)
	fmt.Printf("Determined namespace: %s\n", ns)

	fmt.Println("Creating client...")
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	//TODO: rewrite with watch?
	//TODO: implement waiting for Jobs? (not needed if Helm is used)

	for _, service_name := range services {
		fmt.Printf("Getting '%s' service object...\n", service_name)
		service, err := clientset.CoreV1().Services(ns).Get(service_name, metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}

		set := labels.Set(service.Spec.Selector)

		for {
			fmt.Printf("Getting pods for the '%s' service...\n", service.GetName())
			pods, err := clientset.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: set.AsSelector().String()})
			if err != nil {
				panic(err.Error())
			}

			if len(pods.Items) < 1 {
				fmt.Printf("No pods found for the '%s' service. Retrying...\n", service.GetName())
				time.Sleep(1 * time.Second)
				continue
			}

			fmt.Printf("Checking readiness of the '%s' service pods...\n", service.GetName())

			ready_pod_found := false

			for _, pod := range pods.Items {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == "Ready" && cond.Status == "True" {
						fmt.Printf("%s is ready.\n", pod.GetName())
						ready_pod_found = true
						break
					}
				}
				if ready_pod_found == true {
					break
				}
				fmt.Printf("%s is not ready yet...\n", pod.GetName())
			}
			if ready_pod_found == true {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}
}
