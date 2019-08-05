## Kubernetes Example: leverage CRD to provision Google Cloud SQL DBs on demand

It is a demo repository to add a controller managing a dbbroker CRD to provision in the Google Cloud platform a SQL DB and automatically inject info inside the Pod.

Build making use of Kubebuilder tool

It is described in this Medium article.

# Usage
` export GOOGLE_APPLICATION_CREDENTIALS = "/users/paologallina/Downloads/yourToken.json"`

`GOPATH = $HOME/go`

`make install` to install the CRD

`make run` to let the conteoller run

Now you can already add the annotation to any deployment to trigger the creation of DB and manage their lifecycle
```
dbbroker: managed
dbbroker-db-required: 'true'
```
