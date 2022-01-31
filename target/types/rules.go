package types

// Rule is a string name of a apckage or a service
type Rule string

// Rules is list of packages or services
type Rules []Rule

// File a file to be transfered from local machine to the server
type File struct {
	Owner      string `yaml:"owner,omitempty"`
	Group      string `yaml:"group,omitempty"`
	Mode       int    `yaml:"mode,omitempty"`
	RemotePath string `yaml:"remotepath,omitempty"`
	LocalPath  string `yaml:"localpath,omitempty"`
}

// Host is the basic config for ssh a server
type Host struct {
	Address  string `yaml:"address"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Config the available server config and commands
type Config struct {
	Host    Host   `yaml:"host"`
	Install Rules  `yaml:"install,omitempty"`
	Remove  Rules  `yaml:"remove,omitempty"`
	Run     Rules  `yaml:"run,omitempty"`
	Restart Rules  `yaml:"restart,omitempty"`
	Files   []File `yaml:"transfer_files,omitempty"`
}
