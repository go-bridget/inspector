package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
		return nil, err
	}
	defer client.Close()

	//	socket := os.Getenv("SSH_AUTH_SOCK")

	/* if socket != "" {
		if err := agent.ForwardToRemote(client, socket); err != nil {
			fmt.Println("Error setting up agent forwarding:", err)
		}
	} */

	runCommand := func(command string) string {
		session, err := client.NewSession()
		if err != nil {
			return fmt.Sprintf("ERR: %s", err)
		}
		defer session.Close()

		// if err := agent.RequestAgentForwarding(session); err != nil {
		// 	fmt.Println("Can't enable agent forwarding:", err)
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
		key, err = ioutil.ReadFile(loc)
		if err == nil {
			break
		}
	}
	return key, err
}

func sshConfig() (*ssh.ClientConfig, error) {
	authMethods := []ssh.AuthMethod{}

	// ssh agent
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		fmt.Println("Failed to open SSH_AUTH_SOCK: %v (skipping)", err)
	} else {
		agentClient := agent.NewClient(conn)
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// private key fallback
	key, err := loadSshKey()
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	authMethods = append(authMethods, ssh.PublicKeys(signer))

	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return config, nil
}
