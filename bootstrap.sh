# TODO: make the go version as var via `curl https://golang.org/VERSION?m=text`

wget "https://dl.google.com/go/go1.17.6.linux-amd64.tar.gz"

rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.6.linux-amd64.tar.gz

export PATH=$PATH:/usr/local/go/bin

go version

go mod tidy