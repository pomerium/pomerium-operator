[![pomerium chat](https://img.shields.io/badge/chat-on%20slack-blue.svg?style=flat&logo=slack)](http://slack.pomerium.io)
![Build Status](https://img.shields.io/github/workflow/status/pomerium/pomerium-operator/Default)
[![Go Report Card](https://goreportcard.com/badge/github.com/pomerium/pomerium-operator)](https://goreportcard.com/report/github.com/pomerium/pomerium-operator)
[![Maintainability](https://api.codeclimate.com/v1/badges/df5235a61ea57d8816fc/maintainability)](https://codeclimate.com/github/pomerium/pomerium-operator/maintainability)
[![Documentation](https://godoc.org/github.com/pomerium/pomerium-operator?status.svg)](http://godoc.org/github.com/pomerium/pomerium-operator)
[![LICENSE](https://img.shields.io/github/license/pomerium/pomerium-operator.svg)](https://github.com/pomerium/pomerium-operator/blob/master/LICENSE)
[![codecov](https://img.shields.io/codecov/c/github/pomerium/pomerium-operator.svg?style=flat)](https://codecov.io/gh/pomerium/pomerium-operator)
![Docker Pulls](https://img.shields.io/docker/pulls/pomerium/pomerium-operator)

- [About](#about)
  - [Initial discussion](#initial-discussion)
- [Installing](#installing)
- [Using](#using)
  - [How it works](#how-it-works)
  - [Annotations](#annotations)
  - [Example](#example)
- [Development](#development)
  - [Building](#building)
- [Roadmap](#roadmap)
# About

An operator for running Pomerium on a Kubernetes cluster.

pomerium-operator intends to be the way to automatically configure pomerium based on the state of Ingress, Service and CRD resources in the Kubernetes API Server.  It has aspects of both an Operator and a Controller and in many ways functions as an add-on Ingress Controller.

## Initial discussion 
https://github.com/pomerium/pomerium/issues/273

https://github.com/pomerium/pomerium/issues/425

# Installing
The pomerium operator should be installed with the pomerium helm chart at [https://helm.pomerium.io](https://helm.pomerium.io).

The operator may be run from outside the cluster for development or testing.  In this case, it will use the default configuration at `~/.kube/config`, or you may specify a kubeconfig via the `KUBECONFIG` env var.  Your current context from the config will be used in either case.


# Using

Due to current capabilities, the pomerium-operator is most useful when utilizing [forward auth](https://www.pomerium.io/configuration/#forward-auth).  At this time, you must provide the appropriate annotations
for your ingress controller to have pomerium protect your endpoint.  [Examples](https://www.pomerium.io/recipes/kubernetes.html) can be found in the pomerium documentation.

## How it works

With the operator installed on your cluster (typically via helm chart), it will begin watching `Ingress` and `Service` resources in all namespaces or the
namespace specified by the `namespace` flag.  Following standard ingress controller behavior, pomerium-operator will respond only to resources that match 
the configured `kubernetes.io/ingress.class` and `kubernetes.io/service.class` annotations, or resources without any annotation at all.  

For a given matching resource, pomerium-operator will process all `ingress.pomerium.io/*` annotations and create a policy based on ingress `host` rules (`from` in pomerium policy) and `backend` service names (`to` in pomerium policy).  

Annotations will apply to all rules defined by an ingress resource.

Services _must_ have an `ingress.pomerium.io/from` annotation or they will be ignored as invalid.

## Annotations

pomerium-operator uses a similar syntax for proxying to endpoints based on both Ingress and Service resources.

Policy is set by annotation, as are typical Ingress Controller semantics.

| Key                                             | Description                                                                                                                                                                                                                                            |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| kubernetes.io/ingress.class                     | standard kubernetes ingress class                                                                                                                                                                                                                      |
| kubernetes.io/service.class                     | class for service control. effectively signals pomerium-operator to watch/configure this resource                                                                                                                                                      |
| pomerium.ingress.kubernetes.io/backend-protocol | set backend protocol to http or https. similar to nginx                                                                                                                                                                                                |
| ingress.pomerium.io/[policy_config_key]         | policy_config_key is mapped to a policy configuration of the same name in yaml form. eg, ingress.pomerium.io/allowed_groups is mapped to allowed_groups in the policy block for all service targets in this Ingress. This value should be JSON format. |

## Example

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.pomerium.io/allowed_domains: '["pomerium.io"]'
    nginx.ingress.kubernetes.io/auth-signin: https://forwardauth.pomerium.io/?uri=$scheme://$host$request_uri
    nginx.ingress.kubernetes.io/auth-url: https://forwardauth.pomerium.io/verify?uri=$scheme://$host$request_uri
  labels:
    app: grafana
    chart: grafana-4.3.2
    heritage: Tiller
    release: prometheus
  name: prometheus-grafana
spec:
  rules:
  - host: grafana.pomerium.io
    http:
      paths:
      - backend:
          serviceName: prometheus-grafana
          servicePort: 80
        path: /
```

This ingress:

1. Sets up external auth for nginx-ingress via the `nginx.ingress.kubernetes.io` annotations
2. Maps `grafana.pomerium.io` to the service at `prometheus-grafana`
3. Permits all users from domain `pomerium.io` to access this endpoint

The appropriate policy entry will be generated and injected into the pomerium `ConfigMap`:

```yaml
apiVersion: v1
data:
  config.yaml: |
    policy:
    - from: https://grafana.pomerium.io
      to: https://grafana.pomerium.io
      allowed_domains:
       - pomerium.io
```

# Development

## Building
pomerium-operator utilizes [go-task](https://taskfile.dev/#/) for development related tasks:  

`task build`

# Roadmap 

- [x] Basic CM update functionality.  Provide enough functionality to implement the Forward Auth deployment model.  Basically this is just policy updates being automated and compatible with the current helm chart.  

- [ ] Introduce a mutating webhook that speaks the 3 forward auth dialects and annotates your Ingress for you.  Maybe introduce this configuration via CRD.

- [ ] Get "table stakes" Ingress features into pomerium.  Target model is Inverted Double Ingress or Simple Ingress.  We need cert handling up to snuff, but load balancing and path based routing can be offloaded to a next-hop ingress controller or kube-proxy via Service.  CRD maps which "next-hop" service to use for the IDI model from the ingress class.

- [ ]  Introduce backend load balancing via Endpoint discovery to allow for skipping a second ingress for most configurations.

- [ ]  Allow non-Ingress/Service based policy via CRD.  Helm chart does conversion on the backend.

- [ ]  Pomerium deployment itself is managed by CRD.  The helm chart becomes a wrapper to this CRD.  Move the templating and resource generation logic into pomerium-operator.