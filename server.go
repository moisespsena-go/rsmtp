package rsmtp

import (
	"io"
	"log"
	"errors"
	"io/ioutil"

	"github.com/emersion/go-smtp"
	"github.com/bitly/go-nsq"
	"fmt"
	"github.com/moisespsena/go-error-wrap"
)

type Backend struct{
	Config *Config
	doneCallbacls []func()
}

func NewBackend(config *Config) *Backend {
	return &Backend{Config:config}
}

func (bkd *Backend) Done() {
	for _, d := range bkd.doneCallbacls {
		d()
	}
}

func (bkd *Backend) Start() error {
	nsqd, err := bkd.Config.Nsqd.New()
	if err != nil {
		return errwrap.Wrap(err, "New NSQD")
	}
	err = nsqd.Start(func(message *nsq.Message) error {
		fmt.Println("Received message: %v -> %q", message.ID, string(message.Body))
		return nil
	})

	if err != nil {
		nsqd.Done()
	} else {
		bkd.doneCallbacls = append(bkd.doneCallbacls, func() {
			nsqd.Done()
		})
	}

	return errwrap.Wrap(err, "Start NSQD")
}


func (bkd *Backend) Login(username, password string) (smtp.User, error) {
	usr, ok := bkd.Config.users[username]
	if !ok || usr.Password != password {
		return nil, errors.New("Invalid username or password")
	}
	return &User{}, nil
}

// Require clients to authenticate using SMTP AUTH before sending emails
func (bkd *Backend) AnonymousLogin() (smtp.User, error) {
	return nil, smtp.ErrAuthRequired
}

type User struct{
	AuthNamePassword
	RemoteAcconts []string
	remoteAcconts []*RemoteAccont
}

func (u *User) Send(from string, to []string, r io.Reader) error {
	log.Println("Sending message:", from, to)

	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		log.Println("Data:", string(b))
	}
	return nil
}

func (u *User) Logout() error {
	return nil
}
