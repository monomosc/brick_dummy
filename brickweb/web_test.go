package main_test

import (
	"brick/brickweb/wfe"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateWFE(t *testing.T) {
	w := wfe.New(logrus.New(), MockWFECa{}, MockWFEStorage{}, MockWfeVa{})
	w.BasePath = "https://acme.server"
	w.Handler()
}

const basePath = "https://acme.server"

func TestGetDirectory(t *testing.T) {
	w := wfe.New(logrus.New(), MockWFECa{}, MockWFEStorage{}, MockWfeVa{})
	w.BasePath = basePath
	req := httptest.NewRequest("GET", fmt.Sprintf("%s/dir", basePath), nil)
	rec := httptest.NewRecorder()
	Convey("When Getting Directory", t, func() {
		w.Handler().ServeHTTP(rec, req)
		body, _ := ioutil.ReadAll(rec.Body)
		var dirMap map[string]string
		json.Unmarshal(body, &dirMap)
		Convey("The Directory URLs should match up", func() {
			So(dirMap["newNonce"], ShouldEqual, fmt.Sprintf("%s/new-nonce", basePath))
			So(dirMap["newOrder"], ShouldEqual, fmt.Sprintf("%s/new-order", basePath))
		})
	})
}

func TestGetNonce(t *testing.T) {
	w := wfe.New(logrus.New(), MockWFECa{}, MockWFEStorage{}, MockWfeVa{})
	w.BasePath = basePath
	req := httptest.NewRequest("HEAD", fmt.Sprintf("%s/new-nonce", basePath), nil)
	rec := httptest.NewRecorder()
	Convey("When getting a new nonce", t, func() {
		w.Handler().ServeHTTP(rec, req)
		nonce := rec.Header().Get("replay-nonce")
		Convey("The nonce should be pretty long", func() {
			So(len(nonce), ShouldBeGreaterThan, 20)
		})
	})
}

/*
func TestInvalidNonceShouldReject(t *testing.T) {
	w := wfe.New(logrus.New(), MockWFECa{}, MockWFEStorage{}, make(chan core.VerificationRequest))
	w.BasePath = basePath
	req := httptest.NewRequest("HEAD", fmt.Sprintf("%s/new-nonce", basePath), nil)
	rec := httptest.NewRecorder()
	Convey("When sending request with invalid nonce", t, func() {
		t.SkipNow()
	})
}*/
