package target

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/slack/internal"
	"github.com/slack/target/types"
	"golang.org/x/crypto/ssh"
)

// TODO: test SSH errors response

func TestRemote(t *testing.T) {

	// TEST create New remote
	// unhappy path for wrong config
	_, err := New("localhost", "staff", "", ssh.InsecureIgnoreHostKey(), ssh.Password(""))
	if err == nil {
		t.Errorf("expected error and got nil")
	}

	done := make(chan bool, 1)
	go internal.SetupTestSSH(done)

	// happy path
	r, err := New(internal.LocalAddrString, "staff", "", ssh.InsecureIgnoreHostKey(), ssh.Password(""))
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	err = r.Close()
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// run cmd on closed Remote
	_, err = r.RunCmd("ls", bytes.NewBufferString(""))
	if err == nil {
		t.Errorf("expected error and got nil")
	}

	r, err = New(internal.LocalAddrString, "staff", "", ssh.InsecureIgnoreHostKey(), ssh.Password(""))
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// push files
	remotePath := "testdata/tindex.php"
	localPath := "testdata/index.php"
	err = r.Push(context.Background(), []types.File{
		{
			RemotePath: remotePath,
			LocalPath:  localPath,
			Owner:      "whoami",
			Group:      "whoami",
			Mode:       0644,
		},
	})

	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	remoteBytes, err := os.ReadFile(remotePath)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	localBytes, err := os.ReadFile(localPath)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	if !bytes.Equal(localBytes, remoteBytes) {
		t.Errorf("local files is different than remote file")
	}

	// ensure rule
	p := types.APT{
		Name: "apache2",
	}
	_, err = r.Ensure(p)
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// test Runs
	err = r.Run(context.Background(), []types.Rule{
		"apache2",
	})
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// test Restarts
	err = r.Restart(context.Background(), []types.Rule{
		"apache2",
	})
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// test Installs
	err = r.Install(context.Background(), []types.Rule{
		"apache2",
	})
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	// test Removes
	err = r.Remove(context.Background(), []types.Rule{
		"apache2",
	})
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	err = r.Close()
	if err != nil {
		t.Errorf("expected no errors and got err=%v", err.Error())
	}

	err = r.Close()
	if err == nil {
		t.Errorf("expected error and got nil")
	}

	done <- true
}
