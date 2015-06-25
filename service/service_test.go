package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadServices(t *testing.T) {
	tmpdir := CreateMockFiles()
	defer os.RemoveAll(tmpdir)

	result, err := LoadServices(tmpdir)
	assert.NoError(t, err, "Loading should presente no errors")
	if assert.Len(t, result, NumberOfServiceFiles(), "Number of loaded services is wrong") {
		names := [2]string{result[0].Name, result[1].Name}
		assert.Contains(t, names, "service1", "The name of a service should be 'service1'")
	}
	CleanServices()
}

func TestGetServiceByType(t *testing.T) {
	services = CreateMockServices()

	ws := GetServiceByType("webserver")
	if assert.Len(t, ws, 2, "there should be 2 services with type webserver") {
		assert.Equal(t, "webserver", ws[0].Type, "service type should be webserver")
		img := [2]string{ws[0].Image, ws[1].Image}
		assert.Contains(t, img, "test/tomcat", "images should contain test/tomcat")
	}

	db := GetServiceByType("database")
	if assert.Len(t, db, 1, "there should be 1 services with type database") {
		assert.Equal(t, "database", db[0].Type, "service type should be database")
	}

	ap := GetServiceByType("application")
	assert.Len(t, ap, 0, "there should be 0 services with type application")

	CleanServices()
}

func TestGetServiceByName(t *testing.T) {
	services = CreateMockServices()

	s1, err := GetServiceByName("service2")
	assert.Equal(t, "service2", s1.Name, "service name should be service1")

	_, err = GetServiceByName("pippo")
	assert.Error(t, err, "There should be no service with name 'pippo'")

	CleanServices()
}

func TestGetServiceByImage(t *testing.T) {
	services = CreateMockServices()

	img1, err := GetServiceByImage("test/mysql")
	assert.Equal(t, "test/mysql", img1.Image, "service image should be test/tomcat")

	_, err = GetServiceByImage("test/pippo")
	assert.Error(t, err, "There should be no image 'test/pippo'")

	CleanServices()
}
