# Kubernetes

## RBAC

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ancientt
rules:
  # To create the `ancientt` namespace
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs:
      - get
      - create
  # To get/list, create and delete test runner pods
  - apiGroups: [""]
    resources: ["pods"]
    verbs:
      - get
      - list
      - create
      - delete
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs:
      - "get"
      - "list"
  # Selecting the nodes
  - apiGroups: [""]
    resources: ["nodes"]
    verbs:
      - get
      - list
```
