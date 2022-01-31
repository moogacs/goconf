package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"os/user"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	LocalAddr       = "localhost"
	LocalPort       = 2222
	LocalAddrString = "localhost:2222"
)

type exitStatusMsg struct {
	Status uint32
}

func SetupTestSSH(done chan bool) {
	// Open listen socket
	listener, err := net.Listen("tcp", LocalAddrString)
	if err != nil {
		log.Fatalln(err)
	}

	defer listener.Close()

	for {
		// Accept TCP connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			break
		}

		config := getServerConfig()

		// Perform SSH handshake
		sshConn, newChannels, _, err := ssh.NewServerConn(conn, config)
		if err != nil {
			_ = conn.Close()
			continue
		}

		// Handle new channels
		go handleChannels(newChannels)

		select {
		case <-done:
			sshConn.Close()
			conn.Close()

			return
		default:
		}
	}
}

func getServerConfig() *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalln(err)
	}
	hostKey, err := ssh.NewSignerFromKey(key)
	if err != nil {
		log.Fatalln(err)
	}
	config.AddHostKey(hostKey)
	return config
}

func handleChannels(channels <-chan ssh.NewChannel) {
	for {
		// When a new channel comes in, handle it
		newChannel, ok := <-channels
		if !ok {
			// Connection is closed
			break
		}
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	// Accept all channels. Normally, we would check if it s a "session "channel
	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Println(err)
		return
	}

	for {
		req, ok := <-requests

		if !ok {
			break
		}
		switch req.Type {
		case "subsystem":
			go func() {
				defer channel.Close() // SSH_MSG_CHANNEL_CLOSE
				sftpServer, err := sftp.NewServer(channel)
				if err != nil {
					return
				}
				defer sftpServer.Close()
				_ = sftpServer.Serve()

			}()

			req.Reply(true, nil)

		case "exec":
			u, err := user.Current()
			if err != nil {
				fmt.Println(err)
			}

			toWrite := "test"
			if strings.Contains(string(req.Payload), "-u") {
				toWrite = u.Uid
			}

			if strings.Contains(string(req.Payload), "-g") {
				toWrite = u.Gid
			}

			if req.WantReply {
				_ = req.Reply(true, []byte(toWrite))
				channel.Write([]byte(toWrite))
			}
			channel.SendRequest("exit-status", false, ssh.Marshal(&exitStatusMsg{0}))
			channel.CloseWrite()
			channel.Close()
		default:
			if req.WantReply {
				_ = req.Reply(false, []byte("unsupported request"))
			}
		}
	}
}
