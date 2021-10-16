# Kubebuilder Tutorial
This is a project for learning [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).

This project mainly based on [Kubebuilder Book](https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial.html) but may have additional features.

## Quick Start
### Create a Project
Create a directory, and then run the init command inside of it to initialize a new project. Follows an example.
```shell
$ kubebuilder init --domain example.com --repo github.com/bilalcaliskan/kubebuilder-tutorial
```

**--domain** flag defines the domain of the container registry, while **--repo** defines the github repository url.

### Create an API
Run the following command to create a new API (group/version) as webapp/v1 and the new Kind(CRD) Guestbook on it:
```shell
$ kubebuilder create api --group webapp --version v1 --kind Guestbook
```

> If you press y for Create Resource [y/n] and for Create Controller [y/n] then this will create the files api/v1/guestbook_types.go where the API is defined and the controllers/guestbook_controller.go where the reconciliation business logic is implemented for this Kind(CRD).

### Test It Out
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://github.com/kubernetes-sigs/kind) to get a local cluster for testing, or run against a remote cluster.

> Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster kubectl cluster-info shows).

Install the CRDs into the cluster:
```shell
$ make install
```

Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):
```shell
$ make run
```

### Run It On the Cluster
Build and push your image to the location specified by IMG:
```shell
$ make docker-build docker-push IMG=<some-registry>/<project-name>:tag
```

Deploy the controller to the cluster with image specified by IMG:
```shell
$ make deploy IMG=<some-registry>/<project-name>:tag
```

### Uninstall CRDs
To delete your CRDs from the cluster:
```shell
$ make uninstall
```

### Undeploy controller
Undeploy the controller from cluster:
```shell
$ make undeploy
```

### Implementing defaulting/validating webhooks
If you want to implement [admission webhooks](https://book.kubebuilder.io/reference/admission-webhook.html) for your CRD, the only thing you need to do is to implement the
**Defaulter** and (or) the **Validator** interface.

Kubebuilder takes care of the rest for you, such as:
- Creating the webhook server.
- Ensuring the server has been added in the manager.
- Creating handlers for your webhooks.
- Registering each handler with a path in your server.

First, let’s scaffold the webhooks for our CRD (CronJob). We’ll need to run the following
command with the **--defaulting** and **--programmatic-validation** flags (since our test
project will use defaulting and validating webhooks):
```shell
$ kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

This will scaffold the webhook functions and register your webhook with the manager in your
[main.go](main.go) for you.

## Architectural Concept Diagram
The following diagram will help you get a better idea over the Kubebuilder concepts and architecture.

![Kubebuilder Architectural Diagram](./resources/kubebuilder_architecture.png)

## More On Admission Webhooks
Admission webhooks are HTTP callbacks that receive admission requests, process them and return
admission responses.

Kubernetes provides the following types of admission webhooks:
- **Mutating Admission Webhook:** These can mutate the object while it’s being created or
  updated, before it gets stored. It can be used to default fields in a resource requests,
  e.g. fields in Deployment that are not specified by the user. It can be used to inject
  sidecar containers.
- **Validating Admission Webhook:** These can validate the object while it’s being created
  or updated, before it gets stored. It allows more complex validation than pure schema-based
  validation. e.g. cross-field validation and pod image whitelisting.

The apiserver by default doesn’t authenticate itself to the webhooks. However, if you want
to authenticate the clients, you can configure the apiserver to use basic auth, bearer token,
or a cert to authenticate itself to the webhooks. You can find detailed steps [here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers).

## Markers for Config/Code Generation
KubeBuilder makes use of a tool called controller-gen for generating utility code and
Kubernetes YAML. This code and config generation is controlled by the presence of special
**marker comments** in Go code.

Markers are single-line comments that start with a plus, followed by a marker name,
optionally followed by some marker specific configuration:
```go
// +kubebuilder:validation:Optional
// +kubebuilder:validation:MaxItems=2
// +kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string
```

You can read more about markers [right here](https://book.kubebuilder.io/reference/markers.html).

## Development
This project requires below tools while developing:
- [Golang 1.16](https://golang.org/doc/go1.16)
- [pre-commit](https://pre-commit.com/)
- [golangci-lint](https://golangci-lint.run/usage/install/) - required by [pre-commit](https://pre-commit.com/)
