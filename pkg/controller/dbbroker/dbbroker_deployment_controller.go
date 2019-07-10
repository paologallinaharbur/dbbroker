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
	"github.com/spf13/viper"

	gallocedronev1beta1 "dbbroker/pkg/apis/gallocedrone/v1beta1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r ReconcileDbBrokerDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	instance := &appsv1.Deployment{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if errors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.ObjectMeta.Annotations["dbbroker"] != "managed" {
		return reconcile.Result{}, nil
	}

	//This controller manages this deployment
	//The db is not needed anymore
	if instance.ObjectMeta.Annotations["dbbroker-db-required"] != "true" {
		return r.cleanInfo(instance)
	}

	//The db is needed
	logDeployment("A Deployment with the annotation Changed and we need the db:" + instance.Name + " version: " + instance.ResourceVersion)
	list := &gallocedronev1beta1.DbBrokerList{}
	req, err := labels.NewRequirement("deployment", selection.Equals, append([]string{}, instance.Name))
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.List(context.TODO(), &client.ListOptions{LabelSelector: labels.NewSelector().Add(*req), Namespace: instance.Namespace}, list)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(list.Items) == 0 {
		return r.createDbBroker(instance)
	}
	found := list.Items[0]

	//the dbObject was present checking if the db is initialised
	if found.Status.Initialised != true {
		logDeployment("We found the DB, however it is not initialised yet, it will take care of injecting the info")
		return reconcile.Result{}, nil
	}

	// checking if the info was not deleted
	if !checkIfInfoMissingAndPopulate(&instance.Spec.Template.Spec.Containers[0].Env, found) {
		return reconcile.Result{}, nil // No info was missing
	}

	// Saving the info of the deployment
	err = r.Update(context.TODO(), instance)
	if err != nil {
		log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r ReconcileDbBrokerDeployment) createDbBroker(instance *appsv1.Deployment) (reconcile.Result, error) {

	controller := true
	ToBeCreated := &gallocedronev1beta1.DbBroker{
		ObjectMeta: v1.ObjectMeta{
			Labels:    map[string]string{"deployment": instance.Name},
			Name:      instance.Name + "-" + RandStringBytes(6),
			Namespace: instance.Namespace,
			OwnerReferences: append([]v1.OwnerReference{}, v1.OwnerReference{
				APIVersion: "extensions/v1beta1",
				Kind:       "Deployment",
				Name:       instance.Name,
				UID:        instance.UID,
				Controller: &controller,
			}),
		},
		Spec: gallocedronev1beta1.DbBrokerSpec{
			DeploymentName:      instance.Name,
			DeploymentNameSpace: instance.Namespace,
			ProjectID:           viper.GetString("project.id"),
		},
	}
	log.Printf("Creating dbBroker %s/%s\n", ToBeCreated.Namespace, ToBeCreated.Name)
	err := r.Create(context.TODO(), ToBeCreated)
	return reconcile.Result{}, err
}

func (r ReconcileDbBrokerDeployment) CleanDbObject(name string, namespace string) (reconcile.Result, error) {
	list := &gallocedronev1beta1.DbBrokerList{}
	req, err := labels.NewRequirement("deployment", selection.Equals, append([]string{}, name))
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.List(context.TODO(), &client.ListOptions{LabelSelector: labels.NewSelector().Add(*req), Namespace: namespace}, list)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(list.Items) == 0 {
		return reconcile.Result{}, nil
	}
	for _, db := range list.Items {
		err = r.Delete(context.TODO(), &db)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	logDeployment("Db found and deleted: " + name)
	return reconcile.Result{}, err
}

func checkIfInfoMissingAndPopulate(envVars *[]corev1.EnvVar, dbBroker gallocedronev1beta1.DbBroker) bool {

	size := len(*envVars)
	if size == 0 {
		logDeployment("The Deployment is not containing the Envs, therefore we init the slice")
		*envVars = []corev1.EnvVar{}
	}

	flag := false
	if !isPresent(envVars, "DB_USERNAME") {
		flag = true
		*envVars = append(*envVars, corev1.EnvVar{Name: "DB_USERNAME", Value: dbBroker.Status.Username})
	}

	if !isPresent(envVars, "DB_PASSWORD") {
		flag = true
		*envVars = append(*envVars, corev1.EnvVar{
			Name: "DB_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dbBroker.Name,
					},
					Key:      "DB_PASSWORD",
					Optional: nil,
				},
			},
		})
	}

	if !isPresent(envVars, "DB_PASSWORD_NO_ROOT") {
		flag = true
		*envVars = append(*envVars, corev1.EnvVar{
			Name: "DB_PASSWORD_NO_ROOT",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dbBroker.Name,
					},
					Key:      "DB_PASSWORD_NO_ROOT",
					Optional: nil,
				},
			},
		})
	}

	if !isPresent(envVars, "DB_ENDPOINT") {
		flag = true
		*envVars = append(*envVars, corev1.EnvVar{Name: "DB_ENDPOINT", Value: dbBroker.Status.EndPoint})
	}
	return flag
}

func (r ReconcileDbBrokerDeployment) cleanInfo(instance *appsv1.Deployment) (reconcile.Result, error) {

	envVars := &instance.Spec.Template.Spec.Containers[0].Env

	flag := false
	size := len(*envVars)
	if size == 0 {
		return reconcile.Result{}, nil
	}

	if deleteIfPresent(envVars, "DB_ENDPOINT") {
		flag = true
	}

	if deleteIfPresent(envVars, "DB_PASSWORD") {
		flag = true
	}
	if deleteIfPresent(envVars, "DB_PASSWORD_NO_ROOT") {
		flag = true
	}
	if deleteIfPresent(envVars, "DB_USERNAME") {
		flag = true
	}

	if !flag { //we do not need to update the deploy
		return r.CleanDbObject(instance.Name, instance.Namespace)
	}

	logDeployment("Deleting old resources from Deployment version: " + instance.ResourceVersion)
	err := r.Update(context.TODO(), instance)
	if err != nil {
		log.Error(err)
	}
	return reconcile.Result{}, err
}

func deleteIfPresent(envVars *[]corev1.EnvVar, key string) bool {
	env := *envVars

	for index, e := range env {
		if e.Name == key {
			logDeployment("The env has been found and will be deleted:" + key)
			env = append(env[:index], env[index+1:]...)
			*envVars = env
			return true
		}
	}
	return false
}

func isPresent(envVars *[]corev1.EnvVar, key string) bool {
	for _, e := range *envVars {
		if e.Name == key {
			return true
		}
	}
	logDeployment("The env is missing:" + key)
	return false
}
