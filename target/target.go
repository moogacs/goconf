// Package target contains functionality for *Remote targets
// A Remote target is a target that is connected to using SSH and SFTP (Server)
package target

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/slack/target/types"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

// Remote represents a Remote target, connected to over SSH
type Remote struct {
	addr string

	// auth
	conn     *ssh.Client
	connuser string

	// sudopass is connusers sudo password
	sudopass string

	// the user currently operating as
	activeUser string

	// sftp holds all sftp connections. key is username
	sftp map[string]*sftp.Client
}

type Host interface {
	RunCmd(cmd string, stdin io.Reader) (types.Response, error)
	Push(ctx context.Context, files []types.File) error
	Ensure(p types.APT) (types.StatusCode, error)
	Remove(ctx context.Context, pkgs []types.Rule) error
	Install(ctx context.Context, pkgs []types.Rule) error
	Run(ctx context.Context, pkgs []types.Rule) error
	Restart(ctx context.Context, pkgs []types.Rule) error
	Close() error
}

// New returns a new Remote target from connection details
func New(addr string, user string, sudopass string, hostkeycallback ssh.HostKeyCallback, auths ...ssh.AuthMethod) (Host, error) {

	r := Remote{
		addr:       addr,
		connuser:   user,
		sudopass:   sudopass,
		activeUser: user,
		sftp:       map[string]*sftp.Client{},
	}

	cc := ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: hostkeycallback,
	}

	var err error

	r.conn, err = ssh.Dial("tcp", addr, &cc)
	if err != nil {
		return &r, errors.Wrapf(err, "unable to establish ssh connection to %s", addr)
	}

	return &r, nil
}

// Close closes all underlying connections
func (r *Remote) Close() error {
	for _, c := range r.sftp {
		c.Close()
	}

	return r.conn.Close()
}

// sftpClient returns a sftp client for r.activeUser
// if client does not exist, it will be created
func (r *Remote) sftpClient() (*sftp.Client, error) {
	var c *sftp.Client
	var err error
	var ok bool
	c, ok = r.sftp[r.activeUser]
	if ok {
		return c, nil
	}

	c, err = sftp.NewClient(r.conn)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start sftp connection for %s", r.activeUser)
	}
	r.sftp[r.activeUser] = c
	return c, nil
}

// Run executes cmd on Remote with the currently active user and returns the response.
// Reader stdin is used to add stdin.
func (r *Remote) RunCmd(cmd string, stdin io.Reader) (types.Response, error) {
	return r.run(cmd, stdin)
}

// Push files concurrently using sftp to the target server
func (r *Remote) Push(ctx context.Context, files []types.File) error {
	errs, _ := errgroup.WithContext(ctx)

	sftp, err := r.sftpClient()
	if err != nil {
		return errors.Wrap(err, "could not get sftp client")
	}

	if sftp == nil {
		return errors.Wrap(err, "sftp client is not ready or not found")
	}

	defer sftp.Close()

	for _, cfile := range files {
		file := cfile
		errs.Go(func() error {
			srcFile, err := os.Open(file.LocalPath)
			if err != nil {
				return errors.Wrapf(err, "unable to open file %s", file.LocalPath)
			}
			defer srcFile.Close()

			dstFile, err := sftp.Create(file.RemotePath)
			if err != nil {
				return errors.Wrapf(err, "unable to create file %s", file.RemotePath)
			}
			defer dstFile.Close()

			fmt.Printf("trying to push %s on %s ...\n", file.RemotePath, r.addr)

			n, err := io.Copy(dstFile, srcFile)
			if err != nil {
				return errors.Wrapf(err, "unable to copy to file %s", file.RemotePath)
			}

			st, err := os.Stat(file.LocalPath)
			if err != nil {
				return errors.Wrapf(err, "unable to stat file %s", file.RemotePath)
			}

			if n != st.Size() {
				return fmt.Errorf("wrote %d of %d bytes to file", n, st.Size())
			}

			err = sftp.Chmod(file.RemotePath, fs.FileMode(uint(file.Mode)))
			if err != nil {
				return errors.Wrap(err, "chmod error")
			}

			cmd := fmt.Sprintf("id -u %s", file.Owner)
			res, err := r.run(cmd, bytes.NewBufferString(""))

			if err != nil || !res.Success() {
				return errors.Wrap(errors.New(res.Stderr.String()), "owner error")
			}

			str := strings.ReplaceAll(res.Stdout.String(), "\n", "")
			uid, err := strconv.Atoi(strings.TrimSpace(str))
			if err != nil {
				return errors.Wrap(err, "uid error")
			}

			cmd = fmt.Sprintf("id -g %s", file.Group)
			res, err = r.run(cmd, bytes.NewBufferString(""))
			if err != nil || !res.Success() {
				return errors.Wrap(errors.New(res.Stderr.String()), "group error")
			}

			str = strings.ReplaceAll(res.Stdout.String(), "\n", "")
			gid, err := strconv.Atoi(strings.TrimSpace(str))
			if err != nil {
				return errors.Wrap(err, "gid error")
			}

			err = sftp.Chown(file.RemotePath, uid, gid)
			if err != nil {
				return errors.Wrap(err, "chown error")
			}

			fmt.Printf("%s successfully pushed on %s\n", file.RemotePath, r.addr)
			return nil
		})

	}

	return errs.Wait()
}

// Check checks if package is in the desired state
func (r *Remote) check(p types.APT) (bool, error) {
	cmd := fmt.Sprintf(`dpkg-query -f '${Package}\t${db:Status-Abbrev}\t${Version}\t${Name}' -W %s`, p.Name)

	res, err := r.RunCmd(cmd, bytes.NewBufferString(""))
	if err != nil {
		return false, errors.Wrapf(err, "could not check package status for %s", p.Name)
	}

	if res.ExitStatus == 0 && p.Status == types.StatusNotInstalled {
		return false, nil
	}

	if res.ExitStatus != 0 && p.Status == types.StatusInstalled {
		return false, nil
	}

	var status byte
	// the package info has been returned, so we get the status byte
	stdOutarr := strings.Split(res.Stdout.String(), "\t")
	if len(stdOutarr) > 1 {
		status = stdOutarr[1][1]
	}

	if status != byte(p.Status) {
		return false, nil
	}

	return true, nil
}

// Ensure ensures that the package is in the desired state
func (r *Remote) Ensure(p types.APT) (types.StatusCode, error) {

	ok, err := r.check(p)
	if err != nil {
		return types.StatusFailed, errors.Wrap(err, "ensure check failed")
	}

	if ok && !p.Service() {
		return types.StatusSatisfied, nil
	}

	actions := map[types.Status]string{
		types.StatusInstalled:    "install",
		types.StatusNotInstalled: "purge",

		types.StatusStarted:   "start",
		types.StatusRestarted: "restart",
	}

	cmd := fmt.Sprintf("apt %s %s -y", actions[p.Status], p.Name)

	// on purge remove dependencies
	if p.Status == types.StatusNotInstalled {
		cmd = fmt.Sprintf("apt %s %s -y && apt autoremove -y", actions[p.Status], p.Name)
	}

	if p.Service() {
		cmd = fmt.Sprintf("sudo service %s %s", p.Name, actions[p.Status])
	}
	// TODO for apache check for firewall ->  sudo ufw allow 'Apache'

	res, err := r.RunCmd(cmd, bytes.NewBufferString(""))
	if err != nil || !res.Success() {
		return types.StatusFailed, errors.Wrapf(err, "could not %s package %s", actions[p.Status], p.Name)
	}

	return types.StatusEnforced, nil
}

// Remove removes package and make sure it's in the desired state
func (r *Remote) Remove(ctx context.Context, pkgs []types.Rule) error {
	for _, pkg := range pkgs {
		p := types.APT{
			Name:   string(pkg),
			Status: types.StatusNotInstalled,
			User:   r.activeUser,
		}

		fmt.Printf("trying to remove %s on %s ...\n", pkg, r.addr)

		status, err := r.Ensure(p)
		if err != nil || !status.Success() {
			fmt.Printf("could not remove %s on %s with err=%v\n", pkg, r.addr, err)
			continue
		}

		fmt.Printf("%s is removed on %s\n", pkg, r.addr)
	}

	return nil
}

// Install installs package and make sure it's in the desired state
func (r *Remote) Install(ctx context.Context, pkgs []types.Rule) error {

	for _, pkg := range pkgs {
		p := types.APT{
			Name:   string(pkg),
			Status: types.StatusInstalled,
			User:   r.activeUser,
		}

		fmt.Printf("trying to install %s on %s ...\n", pkg, r.addr)

		status, err := r.Ensure(p)
		if err != nil || !status.Success() {
			fmt.Printf("could not install %s on %s with err=%v\n", pkg, r.addr, err)
			continue
		}

		fmt.Printf("%s is installed on %s\n", pkg, r.addr)
	}

	return nil
}

// Run runs a service and make sure it's in the desired state
func (r *Remote) Run(ctx context.Context, pkgs []types.Rule) error {
	for _, service := range pkgs {
		p := types.APT{
			Name:   string(service),
			Status: types.StatusStarted,
			User:   r.activeUser,
		}

		fmt.Printf("trying to run %s on %s ...\n", service, r.addr)

		status, err := r.Ensure(p)
		if err != nil || !status.Success() {
			fmt.Printf("could not run %s on %s with err=%v\n", service, r.addr, err)
			continue
		}

		fmt.Printf("%s ran on %s\n", service, r.addr)
	}

	return nil
}

// Restart restarts service and make sure it's in the desired state
func (r *Remote) Restart(ctx context.Context, pkgs []types.Rule) error {
	for _, service := range pkgs {
		p := types.APT{
			Name:   string(service),
			Status: types.StatusRestarted,
			User:   r.activeUser,
		}

		fmt.Printf("trying to restart %s on %s ...\n", service, r.addr)

		status, err := r.Ensure(p)
		if err != nil || !status.Success() {
			fmt.Printf("could not restart %s on %s with err=%v\n", service, r.addr, err)
			continue
		}

		fmt.Printf("%s restarted on %s\n", service, r.addr)

	}
	return nil
}

// run runs cmd on remote
func (r *Remote) run(cmd string, stdin io.Reader) (types.Response, error) {
	session, err := r.conn.NewSession()
	resp := types.Response{}

	if err != nil {
		return resp, errors.Wrap(err, "unable to create new session")
	}
	defer session.Close()

	session.Stdout = &resp.Stdout
	session.Stderr = &resp.Stderr
	session.Stdin = stdin

	// TODO: convert it session.StdinPipe() for conccurent  commands
	err = session.Run(cmd)
	if err != nil {
		switch t := err.(type) {
		case *ssh.ExitError:
			resp.ExitStatus = t.Waitmsg.ExitStatus()
		case *ssh.ExitMissingError:
			resp.ExitStatus = -1
		default:
			return resp, errors.Wrap(err, "run of command failed")
		}

	} else {
		resp.ExitStatus = 0
	}

	return resp, nil
}
