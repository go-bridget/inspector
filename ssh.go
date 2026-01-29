package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"unicode"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshRun(server string, columns []Column) ([]Column, error) {
	config, err := sshConfig()
	if err != nil {
		return nil, err
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(server, "22"), config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to remote host: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("debug: error closing ssh client: %v", err)
		}
	}()

	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket != "" {
		if err := agent.ForwardToRemote(client, socket); err != nil {
			fmt.Println("[ForwardToRemote] WARN: error setting up agent forwarding:", err)
		}
	}

	runCommand := func(command string) string {
		session, err := client.NewSession()
		if err != nil {
			return err.Error()
		}
		defer func() {
			if err := session.Close(); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				log.Printf("debug: error closing session: %v", err)
			}
		}()

		// if err := agent.RequestAgentForwarding(session); err != nil {
		//	fmt.Println("[RequestAgentForwarding] WARN: Can't enable agent forwarding:", err)
		// }

		var b bytes.Buffer
		session.Stdout = &b

		if err := session.Run(command); err != nil {
			isEmpty := b.String() == ""
			if !isEmpty {
				b.Write([]byte("\n"))
			}
			b.Write([]byte(err.Error()))
		}
		return strings.TrimRightFunc(b.String(), unicode.IsSpace)
	}

	// columns is already a copy
	for k, v := range columns {
		columns[k].Value = runCommand(v.Command)
	}
	return columns, nil
}

var errKeyNotFound = errors.New("id_rsa file not found")

func loadSshKey() ([]byte, error) {
	locations := []string{
		".ssh/id_rsa",
		path.Join(os.Getenv("HOME"), ".ssh/id_rsa"),
	}
	var (
		key []byte
		err error
	)
	for _, loc := range locations {
		key, err = os.ReadFile(loc)
		if err == nil {
			return key, nil
		}
	}
	return key, errKeyNotFound
}

func sshConfig() (*ssh.ClientConfig, error) {
	authMethods := []ssh.AuthMethod{}

	// ssh agent
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		agentClient := agent.NewKeyring()
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	} else {
		agentClient := agent.NewClient(conn)
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// private key fallback
	key, err := loadSshKey()
	isKeyNotFound := errors.Is(err, errKeyNotFound)
	isKeyFound := !isKeyNotFound

	if isKeyFound {
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return config, nil
}
