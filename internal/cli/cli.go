package cli

import (
	"errors"
	"flag"
	"fmt"
	"hexgen/internal/generate"
	"io"
	"strings"
)

const usage = "Usage: hexgen new NAME --module MODULE --entrypoints http --http stdlib\nOptions: --http stdlib|chi --database postgres --broker rabbitmq|kafka --schedule cron\nThis milestone renders only HTTP with stdlib and no database.\n"

type single struct {
	value string
	set   bool
}

func (s *single) String() string { return s.value }
func (s *single) Set(v string) error {
	if s.set {
		return errors.New("option may only be specified once")
	}
	s.value = v
	s.set = true
	return nil
}
func Run(args []string, stdout, stderr io.Writer, parent string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if len(args) < 2 || args[0] != "new" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	if args[1] == "--help" || args[1] == "-h" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stdout, usage) }
	var module, entry, http, db, broker, schedule single
	fs.Var(&module, "module", "Go module path")
	fs.Var(&entry, "entrypoints", "comma-separated entrypoints")
	fs.Var(&http, "http", "HTTP adapter")
	fs.Var(&db, "database", "database adapter")
	fs.Var(&broker, "broker", "message broker")
	fs.Var(&schedule, "schedule", "scheduler adapter")
	if err := fs.Parse(args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "unexpected positional arguments")
		return 2
	}
	o := generate.Options{Name: args[1], Module: module.value, Entrypoints: strings.Split(entry.value, ","), HTTP: http.value, Database: db.value, Broker: broker.value, Schedule: schedule.value}
	selected := map[string]bool{}
	for _, e := range o.Entrypoints {
		selected[e] = true
	}
	for _, choice := range []struct {
		flag, entry string
		present     bool
	}{{"http", "http", http.set}, {"broker", "consumer", broker.set}, {"schedule", "scheduler", schedule.set}} {
		if choice.present && !selected[choice.entry] {
			fmt.Fprintf(stderr, "--%s requires %s entrypoint\n", choice.flag, choice.entry)
			return 2
		}
	}
	if err := o.Validate(); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	files, err := generate.Render(o)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err = generate.Write(parent, o.Name, files); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Created %s. See %s/README.md for build and run instructions.\n", o.Name, o.Name)
	return 0
}
