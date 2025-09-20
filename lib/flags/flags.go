package flags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/sid-technologies/centurion/lib/types"
)

func ParseArgs(args []string, flags []types.FlagArg) (map[string]any, error) {
	options := make(map[string]any)

	expectedFlags := make(map[string]types.FlagArg)
	for _, flag := range flags {
		expectedFlags[flag.Flag] = flag
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !strings.HasPrefix(arg, "--") {
			return nil, errors.New("unexpected argument: %s, (flags must start with --)", arg)
		}

		// split flag name and value
		var flagName, flagValue string
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flagName = parts[0]
			flagValue = parts[1]
		} else {
			flagName = arg[2:]
			if i+1 >= len(args) {
				return nil, errors.New("missing value for flag: %s", flagName)
			}
			flagValue = args[i+1]
			i++
		}

		opt, exists := expectedFlags[flagName]
		if !exists {
			err_msg := fmt.Sprintf("unexpected flag: --%s", flagName)
			return nil, errors.New(err_msg)
		}

		parsedValue, err := getOptionValue(opt.Type, flagName, flagValue)
		if err != nil {
			return nil, err
		}

		options[opt.Name] = parsedValue
	}

	return options, nil
}

func getOptionValue(flagType string, flagName string, flagValue string) (any, error) {
	var parsedValue any
	var parsedErr error

	switch flagType {
	case "string":
		parsedValue = flagValue
		parsedErr = nil
	case "int":
		parsedValue, parsedErr = strconv.Atoi(flagValue)
	case "float":
		parsedValue, parsedErr = strconv.ParseFloat(flagValue, 64)
	case "bool":
		parsedValue, parsedErr = strconv.ParseBool(flagValue)
	default:
		return nil, errors.New("unsupported flag type %s for flag %s", flagType, flagName)
	}

	if parsedErr != nil {
		return nil, errors.Wrap(parsedErr, "error parsing flag %s with value %s", flagName, flagValue)
	}

	return parsedValue, nil
}

func ValidateRequiredFlags(options []types.FlagArg, providedFlags map[string]string) []types.FlagArg {
	var missingFlags []types.FlagArg

	for _, option := range options {
		if option.Required {
			if _, exists := providedFlags[option.Name]; !exists {
				missingFlags = append(missingFlags, option)
			}
		}
	}

	return missingFlags
}
