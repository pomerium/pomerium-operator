- [About](#about)
  - [Initial discussion](#initial-discussion)
- [Installing](#installing)
- [Spec](#spec)
  - [Annotations](#annotations)
- [Roadmap (tentative)](#roadmap-tentative)
# About

An operator for running Pomerium on a Kubernetes cluster.

pomerium-operator intends to be the way to automatically configure pomerium based on the state of Ingress, Service and CRD resources in the Kubernetes API Server.  It has aspects of both an Operator and a Controller and in many ways functions as an add-on Ingress Controller.

## Initial discussion 
https://github.com/pomerium/pomerium/issues/273
https://github.com/pomerium/pomerium/issues/425

# Installing
TBD helm chart integration

# Spec

## Annotations

pomerium-operator uses a similar syntax for proxying to endpoints based on both Ingress and Service resources.

Policy is set by annotation, as are typical Ingress Controller semantics.

| Key                                             | Description                                                                                                                                                                                                                                            |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| kubernetes.io/ingress.class                     | standard kubernetes ingress class                                                                                                                                                                                                                      |
| kubernetes.io/service.class                     | class for service control. effectively signals pomerium-operator to watch/configure this resource                                                                                                                                                      |
| pomerium.ingress.kubernetes.io/backend-protocol | set backend protocol to http or https. similar to nginx                                                                                                                                                                                                |
| ingress.pomerium.io/[policy_config_key]         | policy_config_key is mapped to a policy configuration of the same name in yaml form. eg, ingress.pomerium.io/allowed_groups is mapped to allowed_groups in the policy block for all service targets in this Ingress. This value should be JSON format. |

# Roadmap (tentative)

1. Basic CM update functionality.  Provide enough functionality to implement the Forward Auth deployment model.  Basically this is just policy updates being automated and compatible with the current helm chart.  

2. Introduce a mutating webhook that speaks the 3 forward auth dialects and annotates your Ingress for you.  Maybe introduce this configuration via CRD.

3. Get "table stakes" Ingress features into pomerium.  Target model is Inverted Double Ingress or Simple Ingress.  We need cert handling up to snuff, but load balancing and path based routing can be offloaded to a next-hop ingress controller or kube-proxy via Service.  CRD maps which "next-hop" service to use for the IDI model from the ingress class.

4.  Introduce backend load balancing via Endpoint discovery to allow for skipping a second ingress for most configurations.

5.  Allow non-Ingress/Service based policy via CRD.  Helm chart does conversion on the backend.

6.  Pomerium deployment itself is managed by CRD.  The helm chart becomes a wrapper to this CRD.  Move the templating and resource generation logic into pomerium-operator.