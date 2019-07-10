/*
Copyright 2019 Paolo.Gallina.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dbbroker

import (
	"context"
	gallocedronev1beta1 "dbbroker/pkg/apis/gallocedrone/v1beta1"
	"dbbroker/pkg/googlecloudsql"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	errorsAPI "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func (r *ReconcileDbBroker) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Fetch the DbBroker dbBrokerInstance
	dbBrokerInstance := &gallocedronev1beta1.DbBroker{}
	err := r.Get(context.TODO(), request.NamespacedName, dbBrokerInstance)
	if err != nil && !errorsAPI.IsNotFound(err) {
		log.Error(err)
		return reconcile.Result{}, err
	}
	if errorsAPI.IsNotFound(err) {
		err := r.cleanOldDb(request.NamespacedName)
		return reconcile.Result{}, err
	}
	if dbBrokerInstance.Status.Initialised {
		return reconcile.Result{}, nil
	}

	logDbController("A dbBroker Changed:" + dbBrokerInstance.Name + "version: " + dbBrokerInstance.ResourceVersion)
	err = r.creteNewDbAndPopulateInfo(dbBrokerInstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	dbBrokerInstance.Status.Initialised = true
	err = r.Update(context.TODO(), dbBrokerInstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDbBroker) cleanOldDb(namespacedName types.NamespacedName) error {
	err := googlecloudsql.DeleteInstances(viper.GetString("project.id"), namespacedName.Name+"-"+namespacedName.Namespace)
	if err != nil {
		return err
	}
	logDbController("DB deleted")
	return nil
}

func (r *ReconcileDbBroker) creteNewDbAndPopulateInfo(dbBrokerInstance *gallocedronev1beta1.DbBroker) error {

	password, err := googlecloudsql.CreateInstances(viper.GetString("project.id"), dbBrokerInstance.Name+"-"+dbBrokerInstance.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	ip, err := googlecloudsql.FetchIp(viper.GetString("project.id"), dbBrokerInstance.Name+"-"+dbBrokerInstance.Namespace, 1)
	if err != nil {
		log.Error(err)
		return err
	}

	username, noRootPassword, err := googlecloudsql.AddUser(viper.GetString("project.id"), dbBrokerInstance.Name+"-"+dbBrokerInstance.Namespace, 1)
	if err != nil {
		log.Error(err)
		return err
	}

	err = r.applySecret(dbBrokerInstance, password, noRootPassword)
	if err != nil {
		log.Error(err)
		return err
	}

	dbBrokerInstance.Status.Username = username
	dbBrokerInstance.Status.EndPoint = ip

	return r.injectInfoDeployment(dbBrokerInstance, password, noRootPassword)
}

func (r *ReconcileDbBroker) injectInfoDeployment(dbBrokerInstance *gallocedronev1beta1.DbBroker, password string, noRootPassword string) error {
	// Fetch the Deployment dbBrokerInstance
	deploymentFound := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: dbBrokerInstance.Spec.DeploymentName, Namespace: dbBrokerInstance.Spec.DeploymentNameSpace}, deploymentFound)
	if err != nil {
		log.Error("we just created a Db but the deployment cannot be fetched")
		return err
	}

	deploymentToBeUpdated := checkIfInfoMissingAndPopulate(&deploymentFound.Spec.Template.Spec.Containers[0].Env, *dbBrokerInstance)

	if !deploymentToBeUpdated {
		return nil
	}

	err = r.Update(context.TODO(), deploymentFound)
	if err != nil {
		log.Error("We tried to inject the variables into the db without success")
		return err
	}
	return nil
}

func (r *ReconcileDbBroker) applySecret(dbBrokerInstance *gallocedronev1beta1.DbBroker, password string, noRootPassword string) error {

	controller := true

	//Creating the secret
	secret := &corev1.Secret{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      dbBrokerInstance.Name,
			Namespace: dbBrokerInstance.Namespace,
			OwnerReferences: append([]v1.OwnerReference{}, v1.OwnerReference{
				APIVersion: "gallocedrone.gallocedrone.io/v1beta1",
				Kind:       "DbBroker",
				Name:       dbBrokerInstance.Name,
				UID:        dbBrokerInstance.UID,
				Controller: &controller,
			}),
		},
		Data:       map[string][]byte{"DB_PASSWORD": []byte(password), "DB_PASSWORD_NO_ROOT": []byte(noRootPassword)},
		StringData: nil,
		Type:       "",
	}

	secretFound := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: dbBrokerInstance.Name, Namespace: dbBrokerInstance.Spec.DeploymentNameSpace}, secretFound)
	if err != nil && !errorsAPI.IsNotFound(err) {
		log.Error("failed to fetch secret")
		return err
	}
	if errorsAPI.IsNotFound(err) {
		logDbController("We did not found the secret, therefore we create it")
		return r.Create(context.TODO(), secret)
	}
	logDbController("We found the secret, therefore we update it")
	if secretFound.Data["DB_PASSWORD"] != nil {
		secret.Data["DB_PASSWORD"] = []byte(secretFound.Data["DB_PASSWORD"])
	}
	return r.Update(context.TODO(), secret)
}

//TODO immutable fields https://github.com/kubernetes/kubernetes/issues/65973
//TODO we should not force the user to attach the db to the first container
//TODO DB without injection
//TODO check if user added many times
//TODO forbid dbbroker deletion to the user, only the controller should do it
//TODO the controller now is running locally, create the image
//TODO error management: delete Not found
