package googlecloudsql

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"math/rand"
	"time"
)

func FetchIp(projectId string, name string, retry int) (string, error) {
	logGoogleCloud("Trying to fetch the IP address of the Db")
	ctx := context.Background()
	service, err := sqladmin.NewService(ctx)
	if err != nil {
		log.Error(err)
		return "", err
	}
	databasesGetCall := service.Instances.Get(projectId, name)
	instance, err := databasesGetCall.Do()
	if err != nil {
		log.Error(err)
		return "", err
	}
	if len(instance.IpAddresses) != 0 {
		logGoogleCloud("IP fetched! " + instance.IpAddresses[0].IpAddress)

		return instance.IpAddresses[0].IpAddress, err
	}

	if retry < 10 {
		log.WithError(err).Warn("The Db is not initialised yet, we should sleep a while")
		time.Sleep(time.Second * 10 * time.Duration(retry))
		return FetchIp(projectId, name, retry+1)

	}
	return "", errors.New("the db is still not available, we give up for now")

}

func CreateInstances(projectId string, name string) (string, error) {
	ctx := context.Background()

	service, err := sqladmin.NewService(ctx)
	if err != nil {
		log.Error(err)
		return "", err
	}
	password := randStringBytes(12)
	database := sqladmin.DatabaseInstance{
		RootPassword:    password,
		Name:            name,
		Project:         projectId,
		DatabaseVersion: "MYSQL_5_7",
		Settings: &sqladmin.Settings{
			Tier: "db-n1-standard-1",
		},
	}

	logGoogleCloud("Creating DB: " + name)
	databasesInsertCall := service.Instances.Insert(projectId, &database)
	_, err = databasesInsertCall.Do()
	if googleapi.IsNotModified(err) {
		logGoogleCloud("Db already existed " + err.Error())
		return "", nil
	} else if err != nil && err.Error() == "googleapi: Error 409: The Cloud SQL instance already exists., instanceAlreadyExists" {
		logGoogleCloud("Db aready existed " + err.Error())
		return "", nil
	} else if err != nil {
		return "", err
	}

	logGoogleCloud("Db created")
	return password, nil
}

func DeleteInstances(projectId string, name string) error {
	ctx := context.Background()

	service, err := sqladmin.NewService(ctx)
	if err != nil {
		log.Error(err)
		return err
	}

	databasesDeleteCall := service.Instances.Delete(projectId, name)
	_, err = databasesDeleteCall.Do()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func AddUser(projectId string, name string, retry int) (string, string, error) {
	ctx := context.Background()
	logGoogleCloud("Adding the user to the db")
	service, err := sqladmin.NewService(ctx)
	if err != nil {
		log.Error(err)
		return "", "", err
	}
	username := randStringBytes(8)
	password := randStringBytes(12)
	_, err = service.Users.Insert(projectId, name, &sqladmin.User{
		Instance: name,
		Name:     username,
		Password: password,
		Project:  projectId,
	}).Do()

	if err == nil {
		return username, password, err
	}
	if retry < 10 {
		log.WithError(err).Warn("The Db is not initialised yet, we should sleep a while")
		time.Sleep(time.Second * 10 * time.Duration(retry))
		return AddUser(projectId, name, retry+1)
	}
	return "", "", errors.New("the db is still not available, we give up for now")
}

func logGoogleCloud(logs string) {
	log.Info("CLOUD_SQL: " + logs)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStringBytes(n int) string {
	b := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
