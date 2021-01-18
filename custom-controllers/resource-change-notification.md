# Responding to Resource Changes through Custom Controllers

Kubernetes applications can request notifications when a particular resource changes, for example adding new pods, deleting services, changes in a configMap and so on. This is useful for applications that require to updates their state, or the state off another application when such changes occur.

For example, say we have an application running in a pod which requires data from a configMap. If the configMap changes, the pods which references this configMap will not be updated; they will still only have the information from the previous version. In order for the pod to get the updated configMap, they need to be recreated. However, kubernetes does not do this automatically.

This is where custom controllers can bridge this gap. The custom controller can request to be notified whenever there is a change in a configMap. Once it does, it could delete the appropriate pods. If these pods were created through a deployment, then Kubernetes will automatically recreate the pods, and attach the updated configMap to the pod.

The workflow will look similar to this:
<img style="background-color: white; margin: auto; display: block;" src="https://www.magalix.com/hs-fs/hubfs/extending%20controllers.png?width=720&name=extending%20controllers.png">

For this to be implemented, there are several components that need to be built.

NOTE: This readme and component samples used in this document come from [Extending the Kubernetes Controller](https://www.magalix.com/blog/extending-the-kubernetes-controller), but were updated to reflect a more current version of Kubernetes. This readme also assumes the reader is well versed with the basics of Kubernetes.

## Creating a Namespace

Though not necessary, it is usually a good idea to create a separate namespace when deploying a group of pods. It allows for different group of applications to have their own set of security, user access, and so on. 

In addition, this custom controller requires additional privileges in order to access the Kubernetes API, which we will cover later. So, having this in its own namespace will ensure these additional rights only apply for deployment in this namespace, and not the entire K8s cluster.

```shell
$ kubectl create namespace test-space
namespace/test-space created
```


## Web Application Components

### Web Server

In our example, the application will be a simple web server application that returns a message whenever someone reaches its endpoint.
```python
from flask import Flask

app = Flask(__name__)
app.config.from_pyfile('/config/config.cfg')


@app.route("/")
def hello():
    return app.config['MSG']


if __name__ == "__main__":
    # Only for debugging while developing
    app.run(host='0.0.0.0', debug=True, port=80)
```
The code reads the message to print from the `/config/config/.cfg` file, which is served through a mounted configMap, which we will show later.

Once the script is created, it needs to be placed in a container through a docker image. As such, the `Dockerfile` will contain the following:
```
FROM tiangolo/uwsgi-nginx-flask:python3.7
COPY ./app /app
```
This Dockerfile assumes that the python script is located in the `app` subdirectory. 

Build the image and then push it to a public Dockerhub account.

```shell
$ docker build -t pivotalosl/docker-test:crd-jason -f Dockerfile .
Sending build context to Docker daemon   7.68kB
Step 1/2 : FROM tiangolo/uwsgi-nginx-flask:python3.7
 ---> da24d18bf4bf
Step 2/2 : COPY ./app /app
 ---> 812857a6ca9a
Removing intermediate container b9c347db3fff
 ---> dca4835aca11
Successfully built dca4835aca11
Successfully tagged pivotalosl/docker-test:crd-jason
$
$ docker push pivotalosl/docker-test:crd-jason
The push refers to repository [docker.io/pivotalosl/docker-test]
1482b2d0da14: Pushed
103b7320095d: Layer already exists
394ec6c8d61d: Layer already exists
c5e393b8a19a: Layer already exists
b3f4557ae183: Layer already exists
9f5b4cdea532: Layer already exists
cd702377e4e5: Layer already exists
aa7af8a465c6: Layer already exists
ef9a7b8862f4: Layer already exists
a1f2f42922b1: Layer already exists
4762552ad7d8: Layer already exists
crd-jason: digest: sha256:aef976b1fecb870c30a1b3080579115c5a7a4d3af2ad6d0750f9c7fcf3ace619 size: 2635
```

### The ConfigMap

Before the application can be deployed, we need to create the ConfigMap, which looks as follows:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: "frontend-config"
  namespace: test-space
  annotations:
    magalix.com/podDeleteMatch: "app=frontend"
data:
  config.cfg:
    MSG="Welcome to Kubernetes!"
```

Things to note in this config map:
1. The annotations section contains a label to help the custom controller identify which pods are using this ConfigMap. This label will be used later on to determine which pods need to be restarted when this ConfigMap changes.
2. The data section contains the message that the web application displays.
3. The configMap is applied in the `namespace: test-space`

Now we apply this definition:
```shell
$ kubectl apply -f configmap.yml
configmap/frontend-config configured
```

### Deploying the Web Application

The web application deployment manifest looks as follows:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  labels:
    app: frontend
  namespace: test-space
spec:
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      namespace: test-space
      labels:
        app: frontend
    spec:
      containers:
      - name: app
        image: pivotalosl/docker-test:crd-jason
        volumeMounts:
        - name: config-vol
          mountPath: /config
      volumes:
      - name: config-vol
        configMap:
          name: frontend-config
```
Two things to note in this deployment:
1. The pods include the `app=frontend` label, which is reference in the configMap annotations section. This is how the custom controller will know which pods to delete.
2. The configMap that was created earlier is mounted onto the `/config` path.
3. The configMap is applied in the `namespace: test-space`

Applying the deployment:
```shell
$ kubectl apply -f deployment.yml
deployment.apps/cuscon configured

$ kubectl get pods --namespace test-space
NAME                        READY   STATUS    RESTARTS   AGE
frontend-7fd8577545-mbqgb   1/1     Running   0          33s
```

Once running, the web application can be tested as follows:
```shell
$ kubectl exec -it frontend-7fd8577545-mbqgb bash
root@frontend-7fd8577545-mbqgb:/app# curl localhost && echo 
Welcome to Kubernetes
root@frontend-7fd8577545-mbqgb:/app#
```

## The Custom Controller

At this stage, changing the configMap message, and applying it, will not affect the frontend pod. The pod will keep the original configMap until it is deleted (at which point K8s would re-create it due to the deployment used to create the pod). K8s does not do this automatically. This is where a custom controller can help.

We need to create a program/script that:
* Continuously watch the API for changes in the ConfigMap.
* When a change is detected, it enumerates all the Pods with the label that matches the one in the ConfigMap annotation and deletes them.

To access the Kubernetes API using raw HTTP calls (no kubectl) you have a number of options:

* Use native client libraries. Currently, Kubernetes officially supports Go, Python, .Net, JavaScript, Java, and Haskell. There is also a number of community-supported libraries in other languages. For more information, please refer to https://kubernetes.io/docs/reference/using-api/client-libraries/
* Use kubectl to create a proxy server. The proxy server listens on a port of your choice and acts as a reverse proxy to the cluster. It relays your API calls to the cluster and it takes care of determining the API it should connect to, the authentication tokens and other required aspects. inister-cluster/access-cluster-api/.

For most instances, the first option is the default choice that should be used. However, it requires more coding than the second option. For the purpose of this example we will use the second option, where the proxy server is loaded as a sidecar. The custom controller container can connect to the API by accessing the kubectl proxy on localhost. This pattern is referred to as the Ambassador pattern.

## API Access

The default user account for each namespace has very limited access to the K8s api. So, instead of giving the default user account the access (which can lead to vulnerabilities), a new service account needs to be created, and then giving the correct access.

The kubectl proxy can then use the token of the new service account to make api calls with, which will be shown later. 

### Service Account

Creating a service account is done as follows:

```shell
$ kubectl create serviceaccount joe --namespace test-space
serviceaccount/joe created
$ kubectl get serviceaccounts --namespace test-space
NAME      SECRETS   AGE
default   1         20h
joe       1         38s
```
Here, a service account called `joe` is created. This will be the account that will have the correct API access

### Roles & RoleBindings
Once the service account is created, we need to create the roles and then bind those roles to the service account.

The roles definitions looks as follows:
```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: watch-configMaps
  namespace: test-space
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: list-delete-pods
  namespace: test-space
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list", "delete"]
```
Two roles are being created:
* First section defines the role access for fetching, watching and listing configMaps resources.
* The second section defines the role access for fetching, watching, listing and deleting pods.

Both of these roles are applied to the `test-space` namespace only.

Though a single role could have been created, it would give the `joe` service account the ability to delete configMaps as well, since verbs are applied to all resources for the role. In our case, only pods need delete access, so two separate roles were created.

The role bindings definitions looks as follows:
```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  creationTimestamp: null
  name: joe-view-configMaps
  namespace: test-space
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: watch-configMaps
subjects:
- kind: ServiceAccount
  name: joe
  namespace: test-space
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  creationTimestamp: null
  name: joe-view-pods
  namespace: test-space
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: list-delete-pods
subjects:
- kind: ServiceAccount
  name: joe
  namespace: test-space
```
Similar to the roles definitions, there are two roles bindings, one to bind each role to the `joe` service account in the `test-space` namespace.

All these four definitions can be put into a single yml file. Then applying the file as follows:
```shell
$ kubectl apply -f custom-controller/rbac.yml
role.rbac.authorization.k8s.io/watch-configMaps created
role.rbac.authorization.k8s.io/list-delete-pods created
rolebinding.rbac.authorization.k8s.io/joe-view-configMaps created
rolebinding.rbac.authorization.k8s.io/joe-view-pods created

$ kubectl get roles --namespace test-space
NAME               AGE
list-delete-pods   32s
watch-configMaps   35s

$ kubectl get rolebindings --namespace test-space
NAME                  AGE
joe-view-configMaps   32s
joe-view-pods         35s
```

With these created, the custom controller code will have appropriate access to the k8s api, as long as it uses the `joe` service account.

### Custom Controller Code

The custom controller looks as follows:
```python
import requests
import os
import json
import logging
import sys

log = logging.getLogger(__name__)
out_hdlr = logging.StreamHandler(sys.stdout)
out_hdlr.setFormatter(logging.Formatter('%(asctime)s %(message)s'))
out_hdlr.setLevel(logging.INFO)
log.addHandler(out_hdlr)
log.setLevel(logging.INFO)

f = open("/var/run/secrets/kubernetes.io/serviceaccount/token", "r")
headers = {"Authorization": "Bearer {}".format(f.read())}
f.close()

base_url = "http://127.0.0.1:8001"


namespace = os.getenv("res_namespace", "default")


def kill_pods(label):
    url = "{}/api/v1/namespaces/{}/pods?labelSelector={}".format(
        base_url, namespace, label)
    r = requests.get(url, headers=headers)
    response = r.json()
    pods = [p['metadata']['name'] for p in response['items']]
    for p in pods:
        url = "{}/api/v1/namespaces/{}/pods/{}".format(base_url, namespace, p)
        r = requests.delete(url)
        if r.status_code == 200:
            log.info("{} was deleted successfully".format(p))
        else:
            log.error("Could not delete {}".format(p))


def event_loop():
    log.info("Starting the service")
    url = '{}/api/v1/watch/namespaces/{}/configmaps'.format(
        base_url, namespace)
    r = requests.get(url, stream=True, headers=headers)
    for line in r.iter_lines():
        obj = json.loads(line)
        event_type = obj['type']
        configmap_name = obj["object"]["metadata"]["name"]
        if "annotations" in obj["object"]["metadata"]:
            if "magalix.com/podDeleteMatch" in obj["object"]["metadata"]['annotations']:
                label = obj["object"]["metadata"]["annotations"]["magalix.com/podDeleteMatch"]
        if event_type == "MODIFIED":
            log.info("Modification detected")
            kill_pods(label)


event_loop()
```
Things of note:
* It opens the `/var/run/secrets/kubernetes.io/serviceaccount/token` file (mounted path fo the `joe` service account) which contains the API token. This is then used to set the bearer token for the API call
* `namespace` is set to the namespace name. We use an environment variable that the Deployment shall possess at runtime. if for any reason the namespace name was not available, we default to “default”.
* `kill_pods` function will find pods with a specific label and then delete them.
* `event_loop` monitors for changes to configMap resources, then it determines if it is the configMap we are interested in. If it is, we fetch the label to use (which was set in the annotation section of the configMap) for deleting specific pods.

This code is loaded into a docker container as such:
```
FROM python:latest
RUN pip install requests
COPY main.py /main.py
ENTRYPOINT [ "python", "/main.py" ]
```

Build and push your container with:
```shell
$ docker build -t pivotalosl/docker-test:cust-ctrl-jason3 -f Dockerfile .
Sending build context to Docker daemon   7.68kB
Step 1/4 : FROM python:latest
 ---> da24d18bf4bf
Step 2/4 : RUN pip install requests
 ---> Using cache
 ---> 051078ba3311
Step 3/4 : COPY main.py /main.py
 ---> 812857a6ca9a
Step 4/4 : ENTRYPOINT [ "python", "/main.py" ]
 ---> Running in b9c347db3fff
Removing intermediate container b9c347db3fff
 ---> dca4835aca11
Successfully built dca4835aca11
Successfully tagged pivotalosl/docker-test:cust-ctrl-jason3

$ docker push pivotalosl/docker-test:cust-ctrl-jason3
The push refers to repository [docker.io/pivotalosl/docker-test]
1482b2d0da14: Pushed
103b7320095d: Layer already exists
394ec6c8d61d: Layer already exists
c5e393b8a19a: Layer already exists
b3f4557ae183: Layer already exists
9f5b4cdea532: Layer already exists
cd702377e4e5: Layer already exists
aa7af8a465c6: Layer already exists
ef9a7b8862f4: Layer already exists
a1f2f42922b1: Layer already exists
4762552ad7d8: Layer already exists
cust-ctrl-jason3: digest: sha256:aef976b1fecb870c30a1b3080579115c5a7a4d3af2ad6d0750f9c7fcf3ace619 size: 2635
```

### Deploying the Custom Controller

The controller's deployment manifest looks as follows:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cuscon
  labels:
    app: cuscon
  namespace: test-space
spec:
  selector:
    matchLabels:
      app: cuscon
  template:
    metadata:
      labels:
        app: cuscon
      namespace: test-space
    spec:
      serviceAccount: joe
      containers:
      - name: proxycontainer
        image: lachlanevenson/k8s-kubectl
        command: ["kubectl","proxy","--port=8001"]
      - name: app
        image: pivotalosl/docker-test:cust-ctrl-jason3
        env:
          - name: res_namespace
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
```
Things of note:
* The deployment is in the `test-space` namespace
* The pods are associated with the `joe` service account. This allows the `proxycontainer` to use that service account instead of `default` so it can access the appropriate k8s endpoints.
* The `proxycontainer` holds the `kubectl` proxy server and will be loaded as a sidecar. So, any other container in this pod can access the k8s api through hostname `localhost` port `8001`
* The `app` container holds the custom controller code. It also sets an environment variable `res_namespace` that holds the namespace that the pod is in. The code reads this environment variable in order to restrict the pod and configMap searches to that namespace.

Apply this deployment:
```shell
$ kubectl apply -f custom-controller/deployment.yml
```

## Testing the Custom Controller

Now that the custom controller and web app are deployed, we can confirm that everything is working by doing a configMap change, applying it and monitoring the pods.

Modify the message of the configMap, which will looks similar to:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: "frontend-config"
  annotations:
    magalix.com/podDeleteMatch: "app=frontend"
data:
  config.cfg: 
    MSG="Welcome to Custom Controllers!"
```

Apply the modified configmap and watch the Flask App Pods:
```shell
$ kubectl apply -f configmap.yml
configmap/frontend-config configured
$ kubectl get pods --watch
NAME                        READY   STATUS    RESTARTS   AGE
cuscon-68cdd77d5c-s45wj     2/2     Running   0          25m
frontend-5989794999-zktqw   1/1     Running   0          6s
NAME                        READY   STATUS        RESTARTS   AGE
frontend-5989794999-zktqw   1/1     Terminating   0          43s
frontend-5989794999-gg294   0/1     Pending       0          1s
frontend-5989794999-gg294   0/1     Pending       0          1s
frontend-5989794999-gg294   0/1     ContainerCreating   0          1s
frontend-5989794999-gg294   1/1     Running             0          3s
frontend-5989794999-zktqw   0/1     Terminating         0          46s
frontend-5989794999-zktqw   0/1     Terminating         0          47s
frontend-5989794999-zktqw   0/1     Terminating         0          47s
```

When the ConfigMap changed, the custom controller automatically detected that change and acted upon it by deleting all the Pods with matching labels. The Flask App deployment automatically started a new Pod to replace the deleted ones. The new Pod is using the updated ConfigMap.

And we can see that the web app is using the new message as follows:
```shell
$ kubectl exec -it frontend-5989794999-gg294 -- bash
root@frontend-5989794999-gg294:/app# curl localhost && echo
Welcome to Custom Controllers!
```