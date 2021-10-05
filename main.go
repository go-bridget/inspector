package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/rodaine/table"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Servers []Server
	Aliases map[string]string
	Columns []Column
}

type Server string

type Column struct {
	Name    string
	Command string
	Value   string
}

func columnNames(columns []Column) []string {
	result := make([]string, len(columns))
	for k, column := range columns {
		result[k] = column.Name
	}
	return result
}

func columnValues(columns []Column) []string {
	result := make([]string, len(columns))
	for k, column := range columns {
		result[k] = column.Value
	}
	return result
}

func ReadConfig(filename string) (*Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := Config{}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func toInterfaceSlice(s1 string, rest []string) []interface{} {
	result := make([]interface{}, len(rest)+1)
	result[0] = s1
	for k, v := range rest {
		result[k+1] = v
	}
	return result
}

func start() error {
	config, err := ReadConfig("inspector.yml")
	if err != nil {
		return err
	}

	var (
		serverWg      sync.WaitGroup
		serverMutex   sync.Mutex
		serverResults = make(map[string][]Column)
	)

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "run":
			commandArgs := os.Args[2:]
			if len(commandArgs) == 0 {
				return errors.New("No command given")
			}
			commandString := strings.Join(commandArgs, " ")
			config.Columns = []Column{
				Column{
					Name:    "Output",
					Command: commandString,
				},
			}

		default:
			if commandString, ok := config.Aliases[os.Args[1]]; ok {
				config.Columns = []Column{
					Column{
						Name:    "Output",
						Command: commandString,
					},
				}
				break
			}
			return errors.New("Invalid parameter error, no such command or alias")
		}
	}

	for _, server := range config.Servers {
		serverWg.Add(1)
		go func(serverName string) {
			defer serverWg.Done()

			columns := make([]Column, len(config.Columns))
			copy(columns, config.Columns[:])

			columns, err := sshRun(serverName, columns)
			if err != nil {
				fmt.Printf("Error for %s: %s\n", serverName, err)
			}

			// fmt.Printf("%s %#v\n", serverName, columns)

			serverMutex.Lock()
			defer serverMutex.Unlock()
			serverResults[serverName] = columns
		}(string(server))
	}

	serverWg.Wait()

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New(toInterfaceSlice("Server", columnNames(config.Columns))...)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, server := range config.Servers {
		serverName := string(server)
		if columns, ok := serverResults[serverName]; ok {
			tbl.AddRow(toInterfaceSlice(serverName, columnValues(columns))...)
		}
	}

	tbl.Print()

	return nil
}

func main() {
	if err := start(); err != nil {
		fmt.Println(err)
	}
}
