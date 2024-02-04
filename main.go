/*
   HTTP server responding to /info endpoint on default port 8080
   To use a different port, pass the env variable PORT to the process
*/
package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"runtime"
)

type ServerInfo struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	IPAddress string `json:"ip_address"`
	Network   string `json:"network"`
}

func getServerInfo() ServerInfo {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	ops := "unknown"
	if osEnv := runtime.GOOS; osEnv != "" {
		ops = osEnv
	}

	ipAddress, network := getIPAddressAndNetwork()

	return ServerInfo{
		Hostname:  hostname,
		OS:        ops,
		IPAddress: ipAddress,
		Network:   network,
	}
}

func getIPAddressAndNetwork() (string, string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown", "unknown"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), ipnet.Network()
			}
		}
	}

	return "unknown", "unknown"
}

func main() {
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		serverInfo := getServerInfo()
		jsonResponse, err := json.Marshal(serverInfo)
		if err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	})

	port := "8080"
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		port = portEnv
	}

	serverAddr := ":" + port
	println("Server listening on", serverAddr)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		panic(err)
	}
}
