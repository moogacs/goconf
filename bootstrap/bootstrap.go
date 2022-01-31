package bootstrap

import (
	"context"
	"fmt"

	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/slack/target"
	"github.com/slack/target/types"
	"golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v3"
)

const (
	tmp = "tmp"
)

type Client struct {
	Configs []types.Config
}

func (bs *Client) Run(cpath, defaultpath string) error {
	configDir, err := os.ReadDir(cpath)
	if err != nil {
		return errors.Wrap(err, "config dir. is not found")
	}

	/**
		TODO : purpose of `tmp`` is to compare the latest state with the new required
	 configuration and produce the difference.

	**/

	// to create tmp dir. to keep state files
	err = CheckDir(tmp)
	if err != nil {
		return err
	}

	defaultConfigs, err := defaultConfig(defaultpath)
	if err != nil {
		return err
	}

	fmt.Println("the following configuration are going to take place:")

	for _, configFile := range configDir {
		configFileContent, err := os.ReadFile(fmt.Sprintf("%s/%s", cpath, configFile.Name()))
		if err != nil {
			return errors.Wrapf(err, "not able to read %s/%s, err=%v", cpath, configFile.Name(), err)
		}

		var config types.Config
		err = yaml.Unmarshal(configFileContent, &config)
		if err != nil {
			return errors.Wrapf(err, "config file %s/%s is corrupted, err=%v", cpath, configFile.Name(), err)
		}

		// adding the defaults
		// TODO : check for dupplications ?
		config.Install = append(config.Install, defaultConfigs.Install...)
		config.Run = append(config.Run, defaultConfigs.Run...)
		config.Files = append(config.Files, defaultConfigs.Files...)

		// validate host
		if config.Host.Address == "" {
			continue
		}

		currentConfigBytes, err := yaml.Marshal(config)
		if err != nil {
			return errors.Wrapf(err, "marshaling config %s/%s", cpath, configFile.Name())
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s.yaml", tmp, config.Host.Address), currentConfigBytes, 0644)
		if err != nil {
			return errors.Wrapf(err, "writing tmp config for %s/%s", cpath, configFile.Name())
		}

		bs.Configs = append(bs.Configs, config)
		fmt.Println(string(currentConfigBytes))
		fmt.Println("----------------")

	}
	return nil
}

func CheckDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0777)
		if err != nil {
			return errors.Wrapf(err, "unable to create tmp directory")
		}
	}

	return nil
}

func (bs *Client) Apply() error {

	for _, config := range bs.Configs {
		addr := fmt.Sprintf("%s:%v", config.Host.Address, config.Host.Port)

		// NOTE : ssh.InsecureIgnoreHostKey() is not production ready and it has to be
		// changed with parsing the correct SSH Keys.
		rmt, err := target.New(addr, config.Host.User, config.Host.Password, ssh.InsecureIgnoreHostKey(), ssh.Password(config.Host.Password))
		if err != nil {
			/*
				 TODO:
					not returning an error and just log it, so that bypass timeouts and wrong config for now
					 as an option  we can have a Health() function to check the servers connections
					 before applying
			*/
			fmt.Printf("could not get new host %v: %v\n", rmt, err)
			continue
		}

		defer rmt.Close()

		// REMOVE pkgs
		err = rmt.Remove(context.Background(), config.Remove)
		if err != nil {
			fmt.Printf("could not remove pkg on %s with err=%v\n", config.Host.Address, err)
		}

		// INSTALL pkgs
		err = rmt.Install(context.Background(), config.Install)
		if err != nil {
			fmt.Printf("could not install pkg on %s with err=%v\n", config.Host.Address, err)
		}

		// RUN services
		err = rmt.Run(context.Background(), config.Run)
		if err != nil {
			fmt.Printf("could not start services on %s with err=%v\n", config.Host.Address, err)
		}

		// RESTART services
		err = rmt.Restart(context.Background(), config.Restart)
		if err != nil {
			fmt.Printf("could not restart services on %s with err=%v\n", config.Host.Address, err)
		}

		// PUSH files
		err = rmt.Push(context.Background(), config.Files)
		if err != nil {
			fmt.Printf("could not push file on %s with err=%v\n", config.Host.Address, err)
		}

		// restart apache
		err = rmt.Restart(context.Background(), []types.Rule{"apache2"})
		if err != nil {
			fmt.Printf("could not push file on %s with err=%v\n", config.Host.Address, err)
		}

		fmt.Printf("%s configuration is done \n----------------------\n", config.Host.Address)
	}

	return nil
}

func defaultConfig(dPath string) (*types.Config, error) {
	dContent, err := os.ReadFile(dPath)
	if err != nil {
		return nil, errors.Wrapf(err, "not able to read %s, err= %v", dPath, err)
	}

	var dConfig types.Config
	err = yaml.Unmarshal(dContent, &dConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "default configs %s is corrupted, err=%v", dPath, err)
	}

	return &dConfig, nil
}
