package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func config(inCluster bool) (*rest.Config, error) {
	if inCluster {
		return rest.InClusterConfig()
	} else {
		kubeconfig := filepath.Join(homeDir(), ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: ./main [agones namespace] [agones fleetname]")
	}
	namespace := os.Args[1]
	fleetName := os.Args[2]

	config, err := config(false)
	if err != nil {
		log.Fatal(err)
	}
	agonescs, err := versioned.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	alloc := agonescs.AllocationV1().GameServerAllocations(namespace)

	http.HandleFunc("/allocate", func(w http.ResponseWriter, r *http.Request) {
		gsa := &allocationv1.GameServerAllocation{
			Spec: allocationv1.GameServerAllocationSpec{
				Required: metav1.LabelSelector{
					MatchLabels: map[string]string{agonesv1.FleetNameLabel: fleetName},
				},
			},
		}
		gsa, err = alloc.Create(gsa)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("failed to allocate: %+v", err)
			return
		}
		addrs := getAddrs(&gsa.Status)
		server := &Server{
			Name:      gsa.Name,
			Addresses: addrs,
		}
		resb, err := json.Marshal(server)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("json marshal err: %+v", err)
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(resb)
	})

	log.Fatal(http.ListenAndServe(":8888", nil))
}

type Server struct {
	Name      string   `json:"name"`
	Addresses AddrPair `json:"addresses"`
}

type AddrPair struct {
	TCP string `json:"tcp"`
	UDP string `json:"udp"`
}

func getAddrs(status *allocationv1.GameServerAllocationStatus) AddrPair {
	pair := AddrPair{}
	for _, port := range status.Ports {
		if port.Name == "tcp" {
			pair.TCP = fmt.Sprintf("%s:%d", status.Address, port.Port)
		}
		if port.Name == "udp" {
			pair.UDP = fmt.Sprintf("%s:%d", status.Address, port.Port)
		}
	}
	return pair
}
