package generate

import (
	"fmt"
	"golang.org/x/mod/module"
	"regexp"
	"strings"
)

type Options struct {
	Name, Module                     string
	Entrypoints                      []string
	HTTP, Database, Broker, Schedule string
}
type File struct {
	Path string
	Data []byte
}

var projectName = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_-]*$")

func device(s string) bool {
	s = strings.ToUpper(strings.Split(s, ".")[0])
	switch s {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	r := []rune(s)
	return len(r) == 4 && (strings.HasPrefix(s, "COM") || strings.HasPrefix(s, "LPT")) && strings.ContainsRune("123456789¹²³", r[3])
}
func validName(s string) error {
	if !projectName.MatchString(s) || device(s) {
		return fmt.Errorf("invalid project name %q", s)
	}
	return nil
}
func (o Options) Validate() error {
	if err := validName(o.Name); err != nil {
		return err
	}
	if err := module.CheckPath(o.Module); err != nil {
		return fmt.Errorf("module: %w", err)
	}
	if len(o.Entrypoints) == 0 {
		return fmt.Errorf("entrypoints are required")
	}
	seen := map[string]bool{}
	for _, e := range o.Entrypoints {
		switch e {
		case "http", "grpc", "consumer", "scheduler":
		default:
			return fmt.Errorf("--entrypoints value %q is invalid; choose http, grpc, consumer, scheduler", e)
		}
		if seen[e] {
			return fmt.Errorf("duplicate entrypoint %q", e)
		}
		seen[e] = true
	}
	if seen["http"] {
		if o.HTTP != "stdlib" && o.HTTP != "chi" {
			return fmt.Errorf("--http value %q is invalid; choose stdlib or chi", o.HTTP)
		}
	} else if o.HTTP != "" {
		return fmt.Errorf("http option requires http entrypoint")
	}
	if o.Database != "" && o.Database != "postgres" {
		return fmt.Errorf("--database value %q is invalid; choose postgres or omit the flag", o.Database)
	}
	if seen["consumer"] {
		if o.Broker != "rabbitmq" && o.Broker != "kafka" {
			return fmt.Errorf("--broker value %q is invalid for consumer; choose rabbitmq or kafka", o.Broker)
		}
	} else if o.Broker != "" {
		return fmt.Errorf("broker requires consumer entrypoint")
	}
	if seen["scheduler"] {
		if o.Schedule != "cron" {
			return fmt.Errorf("--schedule value %q is invalid for scheduler; choose cron", o.Schedule)
		}
	} else if o.Schedule != "" {
		return fmt.Errorf("schedule requires scheduler entrypoint")
	}
	return nil
}
