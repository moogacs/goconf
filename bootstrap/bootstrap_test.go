package bootstrap

import (
	"os"
	"testing"

	"github.com/slack/internal"
	"github.com/slack/target/types"
)

const (
	invalidYaml  = "testdata/invalid_defaults.yaml"
	validYaml    = "testdata/valid_defaults.yaml"
	testTmp      = "testdata/tmp"
	testConfig   = "testdata/config"
	testYaml     = "testdata/config/test.yaml"
	testDefaults = "testdata/defaults.yaml"
)

func TestDefaultConfig(t *testing.T) {
	_, err := defaultConfig("")
	if err == nil {
		t.Errorf("expected an error and got nil")
	}

	_, err = defaultConfig(invalidYaml)
	if err == nil {
		t.Errorf("expected an error and got nil")
	}

	_, err = defaultConfig(validYaml)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}
}

func TestCheckDir(t *testing.T) {

	defer os.Remove(testTmp)
	err := CheckDir(testTmp)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}
}

func TestRun(t *testing.T) {
	b := Client{}
	err := b.Run(testConfig, testDefaults)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	defer os.Remove(testConfig)
	CheckDir(testConfig)
	f, err := os.Create(testYaml)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	defer f.Close()

	f, err = os.Create(testDefaults)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	defer f.Close()
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	err = b.Run(testConfig, testDefaults)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// TODO create valid file to test the tmp creation
}

func TestApply(t *testing.T) {
	c := Client{
		Configs: []types.Config{
			{
				Host: types.Host{
					Address: internal.LocalAddr,
					Port:    internal.LocalPort,
				},
			},
		},
	}

	done := make(chan bool, 1)
	go internal.SetupTestSSH(done)

	err := c.Apply()
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}
}
