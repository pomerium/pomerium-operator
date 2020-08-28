# Dependency Upgrade Notes

Due to the fragility and interdependencies between kubernetes packages, upgrades must be undertaken carefully.

## Typical

- Upgrade controller-runtime
- Upgrade client-go to match a kubernetes version
- Upgrade/downgrade depdendencies that may have thrashed (typically, apimachinery)


### Example

```bash
go get -u sigs.k8s.io/controller-runtime@v0.5.9
go get -u k8s.io/client-go@kubernetes-1.17.11
go get -u k8s.io/apimachinery@kubernetes-1.17.11
```

## History

### 1.16->1.17

* gnostic break at v0.4.1 is still a problem, but now half the ecosystem has fixed it.  Full fix anticipated in 1.19.
* logr introduced a breaking change but is not systemically supported in controller-runtime until v0.7 / kube-1.19

```bash
go get sigs.k8s.io/controller-runtime@v0.5.9
go get k8s.io/client-go@kubernetes-1.17.11
go get github.com/googleapis/gnostic@v0.4.0
go get k8s.io/apimachinery@kubernetes-1.17.11
go get github.com/go-logr/logr@v0.1.0
```