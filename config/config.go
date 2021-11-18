/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/trace"

	"github.com/sirupsen/logrus"
)

//OpentracingConfig determines the Settings used for the opentracing Implementation
type OpentracingConfig struct {
	Enable bool `json:"enable"`
}

type PrometheusConfig struct {
	Port int `json:"port"`
}

type FileLogConfig struct {
	Enable   bool   `json:"enable"`
	Location string `json:"location"`
}

func ReadJSON(file string, configData interface{}) error {
	cfg, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(cfg, configData)
}

//StartProcessTracing returns a Closer to stop ProcessTracing
func StartProcessTracing() io.Closer {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	err = trace.Start(f)
	if err != nil {
		panic(err)
	}
	return closerFunc(func() {
		trace.Stop()
	})
}

type closerFunc func()

func (f closerFunc) Close() error {
	f()
	return nil
}

var writer io.Writer = os.Stdout
var stage string = "DEV"

func SetWriter(fileLogging FileLogConfig) {
	if fileLogging.Enable {
		f, err := os.Create(fileLogging.Location)
		if err != nil {
			panic(fmt.Errorf("Could not openFile, because of %v", err))
		}
		writer = io.MultiWriter(os.Stdout, f)
	} else {
		writer = os.Stdout
	}
}
func SetStage(_stage string) {
	stage = _stage
}

func MakeStandardLogger(program string, component string, json bool) logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = writer

	if json {
		logger.Formatter = &logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "Timestamp",
				logrus.FieldKeyLevel: "Level",
				logrus.FieldKeyMsg:   "MessageTemplate",
			},
			TimestampFormat: "2006-01-02T15:04:05.000-07:00",
			DataKey:         "Properties",
		}
	} else {
		logger.Formatter = &logrus.TextFormatter{DisableColors: true}
	}
	logger.Level = logrus.DebugLevel
	return logger.WithField("comp", component).WithField("App", "Brick").WithField("Program", program).WithField("Stage", stage)
}
