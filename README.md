# goconf
The rudimentary golang config tool purpose is to config PHP servers via SS


 ## Packages

 `bootstrap`  is the package responsible for initating, running and applying the configurations.

 `cmd` is the exec package, it has:
   - `config` directory which responsible about the configuration for every server.
   - `server` holds the files which needs to be added to th specified server. it consists of directories `{dir name}` is server name.
   - `tmp` is generated directory for the produced condiguration per server. #TODO is to compare the latest state with the new required configuration and produce the difference.
   - `defaults.yaml` is the default configuration for `PHP` server

`internal` has the testing utils for creating ssh server for testing purpose.

 `target` has the remote implementation for accessing servers via `ssh`, `sftp`  for exec and transfering files.


### Avaliable rules:  `install`, `remove`, `run`, `restart`, `transfer_files`
<br>

## Config file

the config has to be a `yaml` file  as the mentioned structure at the bottom and
has to be added in the `cmd/config` directory

Note: for the config files there is no need to add the PHP needed configs. it's already added in order to not cause duplications.

for example for just displaying the `index.php` see config at `cmd/config/54.92.218.144.yaml`

```
host:
    address: server ip
    port: 22
    user: root
    password:  password

install:
  - golang-go
  - apache2

run:
  - apache2

restart:
  - apache2

transfer_files:
  - owner: root
    group: root
    mode: 0644
    localpath: server/server-1-ip/test.txt
    remotepath: /root/hello.txt
  - owner: root
    group: root
    mode: 0644
    localpath: server/ip/test2.txt
    remotepath: /root/hello2.txt
```

<br/>

## Run the tool

```
cd cmd
go run main.go
```

OR 

```
cd cmd
go run main.go
```


on Linux
```
./slack-challenge
```

on OSX 
```
./slack-challenge-amd64
```