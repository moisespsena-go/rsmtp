package rsmtp

import (
	"os/exec"
	"github.com/moisespsena/go-error-wrap"
	"path/filepath"
	"strings"
	"fmt"
	"github.com/bitly/go-nsq"
	"sync"
	"time"
	"syscall"
)

type NsqdConfig struct {
	ExecArgs []string
	ExecAdmin bool
	ExecAdminArgs []string
	Addr string
	Topic string
}

func (n *NsqdConfig) New() (nsqd *Nsqd, err error) {
	if n.Topic == "" {
		return nil, fmt.Errorf("nsqd.topic is empty.")
	}
	if n.Addr == "" {
		return n.newFromBin()
	}
	return &Nsqd{Topic:n.Topic, Addr:n.Addr}, nil
}

func (n *NsqdConfig) newFromBin() (nsqd *Nsqd, err error)  {
	var cmds []*exec.Cmd

	if n.Addr == "" {
		for _, arg := range n.ExecArgs {
			if strings.HasPrefix(arg, "--lookupd-tcp-address=") {
				n.Addr = strings.TrimPrefix(arg, "--lookupd-tcp-address=")
				break
			}
		}
	}

	if n.Addr == "" {
		return nil, fmt.Errorf("invalid NSQD addr.")
	}

	binDir := filepath.Dir(n.ExecArgs[0])

	e := exec.Command(filepath.Join(binDir, "nsqlookupd"))
	cmds = append(cmds, e)

	e = exec.Command(n.ExecArgs[0], n.ExecArgs[1:]...)
	cmds = append(cmds, e)

	e = exec.Command(filepath.Join(binDir, "nsqadmin"), n.ExecAdminArgs...)
	cmds = append(cmds, e)
	return &Nsqd{Topic:n.Topic, Addr:n.Addr,Cmds:cmds}, nil
}

type Nsqd struct {
	Topic string
	Addr string
	Cmds []*exec.Cmd
	Producer *nsq.Producer
	doneCallbacks []func()
}

func (n *Nsqd) Done() {
	for _, cmd := range n.Cmds {
		if cmd.Process != nil && cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
			cmd.Process.Kill()
		}
	}

	for _, done := range n.doneCallbacks {
		done()
	}
}

func (n *Nsqd) Start(callback func(message *nsq.Message) error) (err error) {
	defer func() {
		if err != nil {
			n.Done()
		}
	}()
	for _, cmd := range n.Cmds {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Pdeathsig: syscall.SIGTERM,
		}
		err = cmd.Start()
		if err != nil {
			err = errwrap.Wrap(err, "CMD %q --> args: %v", cmd.Path, cmd.Args)
			return err
		}
		<- time.After(time.Second)
	}

	w, err := nsq.NewProducer(n.Addr, nsq.NewConfig())
	if err != nil {
		err = errwrap.Wrap(err, "New Producer %q", n.Addr)
		for _, cmd := range n.Cmds {
			err2 := cmd.Process.Kill()
			if err2 != nil {
				err = errwrap.Wrap(err2, err)
			}
		}
		return err
	}

	n.Producer = w
	go w.Stop()
	<- time.After(time.Second)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	q, _ := nsq.NewConsumer(n.Topic, "ch", nsq.NewConfig())
	q.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		err = callback(message)
		wg.Done()
		return err
	}))
	err = q.ConnectToNSQD(n.Addr)
	if err != nil {
		return errwrap.Wrap(err, "Could not connect to %q.", n.Addr)
	}

	go wg.Wait()
	return nil
}