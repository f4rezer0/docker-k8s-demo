## instructions

Init the golang project:
```bash
$ go mod init serverinfo
```
Create the `main.go`:
```bash
$ cat > main.go << 'EOF'
package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
        "runtime"
)

type ServerInfo struct {
	Hostname   string   `json:"hostname"`
	OS         string   `json:"os"`
	IPAddress  string   `json:"ip_address"`
	Network    string   `json:"network"`
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
		Hostname:   hostname,
		OS:         ops,
		IPAddress:  ipAddress,
		Network:    network,
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
EOF
```

Create the `.gitignore`:
```bash
$ cat > .gitignore << 'EOF'
serverinfo
EOF
```

Build and test locally:
```bash
$ go build -o serverinfo .
$ ./serverinfo
```
Then you can open a new terminal and call:
```bash
$ curl http://localhost:8080/info | jq
```

Create the `Dockerfile`:
```bash
$ cat > Dockerfile << 'EOF'
# build stage
FROM golang:1.21 AS build-stage

WORKDIR /build
COPY . /build/

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o serverinfo .

RUN chmod +x serverinfo

# final stage ~~~~~~~~~~~~~~~~~~~
FROM busybox:latest

COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=build-stage /build/serverinfo ./

RUN addgroup -S appgroup && adduser -S --no-create-home appuser -G appgroup
USER appuser

CMD ["./serverinfo"]
EOF
```

Allow permissions on the created files: 
```bash
ls -a | xargs -I {} chmod 1777 {}
```

Build the docker image:
```bash
$ docker build -t alessandroargentieri/serverinfo:v0.0.1 -t alessandroargentieri/serverinfo:latest .
```

Verify the image is present:
```bash
$ docker images | grep serverinfo
```

Analyse its layers:
```bash
$ docker history alessandroargentieri/serverinfo:latest
```

Push on the DockerHub registry:
```bash
$ docker push alessandroargentieri/serverinfo:v0.0.1
$ docker push alessandroargentieri/serverinfo:latest
```

If you need to pull from another machine run:
```bash
$ docker pull alessandroargentieri/serverinfo:latest
```

If you want to run the container from that image (either if you don't pull first it's automatically pulled if needed):
```bash
$ docker run -it -d -p 8082:8081 -e PORT=8081 --name=serverinfo alessandroargentieri/serverinfo:latest 
```

You can check the logs from the `serverinfo` running container with:
```bash
$ docker logs serverinfo
```

If you have the previous process running locally on port 8080, now you can verify both the answers (from the local running 
`serverinfo` and the one running in docker and exposed to the local machine on port 8082) by executing:
```bash
$ curl http://localhost:8080 | jq
{
  "hostname": "Alessandros-MacBook-Pro.local",
  "os": "darwin",
  "ip_address": "192.168.1.128",
  "network": "ip+net"
}

$ curl http://localhost:8082 | jq
{
  "hostname": "88ae85803494",
  "os": "linux",
  "ip_address": "172.17.0.4",
  "network": "ip+net"
}
```
You can verify the IP assigned from docker to the given cluster with:
```bash
$ docker inspect --format="{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}" serverinfo
172.17.0.4
```

You can try to access the container (you need to comment the `adduser` into the `Dockerfile` and rebuild the image to allow root 
access to the container):
```bash
$ docker exec -it serverinfo sh
# write 'exit' to esc
```

If you want to execute some other operation from the container you can do:
```bash
$ docker exec -it serverinfo env
```

If you want to enter the container and modify its state and create a new snapshot (a new image) from the current state you can 
do:
```bash
$ docker commit serverinfo alessandroargentieri/serverinfo:v0.0.2
$ docker push alessandroargentieri/serverinfo:v0.0.2
```
To stop and re-run the `serverinfo` container:
```bash
$ docker stop serverinfo
$ docker start serverinfo
```
To force delete the running container:
```bash
$ docker rm -f serverinfo
```
To delete the local image:
```bash
$ docker rmi alessandroargentieri/serverinfo:v0.0.1
# or alternatively:
#   docker image rm alessandroargentieri/serverinfo:v0.0.1 
```
## Bonus: use the container in a kubernetes cluster

You can reuse images wherever you want. In this example we're going to launch our `serverinfo` application in a managed Kubernetes cluster with Civo:
If you have civo CLI configured you can create a cluster with:
```bash
$ civo kubernetes create serverinfo-example --nodes=3 --region=lon1
```
When the cluster is up-and-running you can download the kubeconfig and use it to configure your `kubectl` to access the brand new cluster:
```bash
$ civo kubernetes config serverinfo-example > ~/.kube/config_serverinfo-example
$ export KUBECONFIG=/Users/alessandroargentieri/.kube/config_serverinfo-example
```
You can create a `Deployment` kubernetes object with 10 replicas of the `alessandroargentieri/serverinfo` image:
```bash
$ cat > serverinfo-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: serverinfo
spec:
  replicas: 10
  selector:
    matchLabels:
      app: serverinfo
  template:
    metadata:
      labels:
        app: serverinfo
    spec:
      containers:
        - name: serverinfo
          image: alessandroargentieri/serverinfo:latest
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: serverinfo-lb
spec:
  selector:
    app: serverinfo
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
EOF
$ kubectl apply -f serverinfo-deployment.yaml
```

If you want to list all pods in the `default` namespace according to the node in which they're deployed you can:
```bash
$ kubectl get nodes --no-headers | cut -d ' ' -f 1 | xargs -I {} bash -c "echo;echo node {};  kubectl get pods -o wide --field-selector spec.nodeName={}"
```
## Bonus: use the container in a local distro of a kubernetes cluster

We want to use `k3d` as local kubernetes distribution.

To install it:
```bash
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
```
Once installed, to create a new cluster:
```bash
 $ k3d cluster create serverinfo --agents 3
```
Because no real LoadBalancer is created, let's port forward localhost on the k8s service:
```bash
$ kubectl port-forward service/serverinfo-lb 8080:80
```
To test it:
```bash
$ curl http://localhost:8080/info | jq
```

Let's ssh into one of the three agent nodes:
```bash
$ docker exec -it k3d-serverinfo-agent-0 sh
```
Inside the node we don't have docker installed but we can use `crictl` CLI to query the `containerd` images and containers:
```bash
/ # crictl images
/ # crictl ps
```
Let's get the ID of one of the `serverinfo` containers running in the node and ssh into it!
```bash
/ # crictl exec -it 788847871628b sh
```
To delete the cluster:
```bash
$ k3d cluster delete serverinfo
```
