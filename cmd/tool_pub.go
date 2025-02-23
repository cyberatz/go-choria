// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type tPubCommand struct {
	command
	topic string
	input string
}

func (p *tPubCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		p.cmd = tool.Cmd().Command("pub", "Publish to middleware topics")
		p.cmd.Arg("topic", "The topic to subscribe to").Required().StringVar(&p.topic)
		p.cmd.Flag("input", "File contents to publish, else STDIN").ExistingFileVar(&p.input)
	}

	return nil
}

func (p *tPubCommand) Configure() error {
	return commonConfigure()
}

func (p *tPubCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	log := c.Logger("pub")
	conn, err := c.NewConnector(ctx, c.MiddlewareServers, c.Certname(), log)
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}

	input := os.Stdin

	if p.input != "" {
		input, err = os.Open(p.input)
		if err != nil {
			return fmt.Errorf("cannot open input %s: %s", p.input, err)
		}
		defer input.Close()
	}

	body, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("could not read from input: %s", err)
	}

	if len(body) == 0 {
		return fmt.Errorf("read 0 bytes from input")
	}

	err = conn.PublishRaw(p.topic, body)
	if err != nil {
		return fmt.Errorf("could not publish: %s", err)
	}

	fmt.Printf("Published %d bytes to %s on %s\n", len(body), p.topic, conn.ConnectedServer())

	conn.Close()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tPubCommand{})
}
