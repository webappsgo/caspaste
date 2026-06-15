
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func exitOnError(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}

// getEnvVar attempts to read a CASPB_* environment variable.
// Returns the value and true if found, empty string and false otherwise.
func getEnvVar(name string) (string, bool) {
	// Convert flag name to environment variable format
	// Example: "db-driver" -> "DB_DRIVER"
	envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	if val := os.Getenv("CASPB_" + envName); val != "" {
		return val, true
	}

	return "", false
}

type variable struct {
	name        string
	cliFlagName string

	preHook func(string) (string, error)

	value        interface{}
	valueDefault string
	required     bool
	usage        string
}

type CLI struct {
	version string

	vars []variable
}

type FlagOptions struct {
	Required bool
	PreHook  func(string) (string, error)
}

func New(version string) *CLI {
	return &CLI{
		version: version,

		vars: []variable{},
	}
}

func (c *CLI) addVar(name string, value interface{}, defValue string, usage string, opts *FlagOptions) {
	if name == "" {
		panic("cli: add variable: variable name could not be empty")
	}

	if usage == "" {
		panic("cli: flag \"" + name + "\" has empty \"usage\" field")
	}

	if opts == nil {
		opts = &FlagOptions{}
	}

	c.vars = append(c.vars, variable{
		name:        name,
		cliFlagName: "-" + name,

		preHook: opts.PreHook,

		value:        value,
		valueDefault: defValue,
		required:     opts.Required,
		usage:        usage,
	})
}

func (c *CLI) AddStringVar(name, defValue string, usage string, opts *FlagOptions) *string {
	if opts != nil {
		if opts.PreHook != nil {
			var err error
			defValue, err = opts.PreHook(defValue)
			if err != nil {
				panic("cli: add duration variable \"" + name + "\": " + err.Error())
			}
		}
	}

	val := &defValue
	c.addVar(name, val, defValue, usage, opts)
	return val
}

func (c *CLI) AddBoolVar(name string, usage string) *bool {
	valVar := false
	val := &valVar
	c.addVar(name, val, "", usage, nil)
	return val
}

func (c *CLI) AddIntVar(name string, defValue int, usage string, opts *FlagOptions) *int {
	val := &defValue
	c.addVar(name, val, strconv.Itoa(defValue), usage, opts)
	return val
}

func (c *CLI) AddUintVar(name string, defValue uint, usage string, opts *FlagOptions) *uint {
	val := &defValue
	c.addVar(name, val, strconv.FormatUint(uint64(defValue), 10), usage, opts)
	return val
}

func (c *CLI) AddDurationVar(name, defValue string, usage string, opts *FlagOptions) *time.Duration {
	if opts != nil {
		if opts.PreHook != nil {
			var err error
			defValue, err = opts.PreHook(defValue)
			if err != nil {
				panic("cli: add duration variable \"" + name + "\": " + err.Error())
			}
		}
	}

	valDuration, err := ParseDuration(defValue)
	if err != nil {
		panic("cli: add duration variable \"" + name + "\": " + err.Error())
	}

	val := &valDuration
	c.addVar(name, val, defValue, usage, opts)
	return val
}

func writeVar(val string, to interface{}, preHook func(string) (string, error)) error {
	if preHook != nil {
		var err error
		val, err = preHook(val)
		if err != nil {
			return err
		}
	}

	switch to := to.(type) {
	case *string:
		*to = val

	case *int:
		val, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		*to = val

	case *bool:
		val := true
		*to = val

	case *uint:
		val, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		*to = uint(val)

	case *time.Duration:
		val, err := ParseDuration(val)
		if err != nil {
			return err
		}
		*to = val

	default:
		panic("cli: write variable: unknown \"to\" argument type")
	}

	return nil
}

func (c *CLI) printVersion() {
	fmt.Println(c.version)
	os.Exit(0)
}

func (c *CLI) printHelp() {
	// Search for the longest flag and required flags list.
	var maxFlagSize int
	var reqFlags string

	for _, v := range c.vars {
		flagSize := len(v.cliFlagName)
		if flagSize > maxFlagSize {
			maxFlagSize = flagSize
		}

		if v.required {
			reqFlags += "[" + v.cliFlagName + "] "
		}
	}

	// Print help
	fmt.Println("Usage:", os.Args[0], reqFlags+"[OPTION]...")
	fmt.Println("")

	for _, v := range c.vars {
		var spaces string
		for i := 0; i < maxFlagSize-len(v.cliFlagName)+2; i++ {
			spaces += " "
		}

		var defaultStr string
		if v.valueDefault != "" {
			defaultStr = " (default: " + v.valueDefault + ")"
		}

		fmt.Println(" ", v.cliFlagName, spaces, v.usage+defaultStr)
	}

	fmt.Println()
	fmt.Println("  -version   Display version and exit.")
	fmt.Println("  -help      Display this help and exit.")

	os.Exit(0)
}

// normalizeFlag converts --flag to -flag for backward compatibility
func normalizeFlag(arg string) string {
	if strings.HasPrefix(arg, "--") {
		return strings.TrimPrefix(arg, "-")
	}
	return arg
}

func (c *CLI) Parse() {
	// The name of variables that were read from environment variables or CLI flags.
	// Used to check if "required" flags are present.
	readVars := make(map[string]struct{})

	// Read variables from environment variables first
	// Read CASPB_* environment variables
	for i := range c.vars {
		v := &c.vars[i]
		if envVal, found := getEnvVar(v.name); found {
			err := writeVar(envVal, v.value, v.preHook)
			if err != nil {
				exitOnError("read environment variable for \"" + v.name + "\": " + err.Error())
			}
			readVars[v.name] = struct{}{}
		}
	}

	// Read variables from CLI flags (these override environment variables)
	{
		alreadyRead := make(map[string]struct{})

		var varInProgress *variable
		for _, arg := range os.Args[1:] {
			if varInProgress == nil {
				// Normalize --flag to -flag for backward compatibility
				normalizedArg := normalizeFlag(arg)

				switch normalizedArg {
				case "-version":
					c.printVersion()

				case "-help":
					c.printHelp()
				}

				_, exist := alreadyRead[normalizedArg]
				if exist {
					exitOnError("flag \"" + normalizedArg + "\" occurs twice")
				}

				ok := false
				for _, v := range c.vars {
					if v.cliFlagName == normalizedArg {
						switch v.value.(type) {
						case *bool:
							// Boolean flag - set to true immediately
							*(v.value.(*bool)) = true
						default:
							varInProgress = &v
						}

						alreadyRead[normalizedArg] = struct{}{}
						readVars[v.name] = struct{}{}

						ok = true
						break
					}
				}

				if !ok {
					exitOnError("unknown flag \"" + arg + "\"")
				}

			} else {
				err := writeVar(arg, varInProgress.value, varInProgress.preHook)
				if err != nil {
					exitOnError("read \"" + varInProgress.cliFlagName + "\" flag: " + err.Error())
				}

				varInProgress = nil
			}
		}

		if varInProgress != nil {
			exitOnError("no value for \"" + varInProgress.cliFlagName + "\" flag")
		}
	}

	// Check required variables
	for _, v := range c.vars {
		if v.required {
			_, ok := readVars[v.name]
			if !ok {
				exitOnError("\"" + v.cliFlagName + "\" flag is missing")
			}
		}
	}
}
