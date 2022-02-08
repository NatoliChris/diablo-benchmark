package main


import (
	"fmt"
	"strings"
)


type shortOption struct {
	name   byte
	value  bool
	cb     func(string) error
}

type longOption struct {
	name   string
	value  bool
	cb     func(string) error
}


func parseLongOption(arg string, longs []longOption) (func(string) error, error) {
	var optname, optval string
	var hasVal bool
	var i, idx int
	var err error

	idx = strings.Index(arg, "=")

	if idx > 0 {
		hasVal = true
		optname = arg[2:idx]
		optval = arg[(idx+1):]
	} else {
		hasVal = false
		optname = arg[2:]
	}

	for i = range longs {

		if longs[i].name != optname {
			continue
		}

		if hasVal {
			if longs[i].value {
				err = longs[i].cb(optval)

				if err != nil {
					return nil, fmt.Errorf("invalid " +
						"value for '--%s': %s",
						optname, err.Error())
				}

				return nil, nil
			} else {
				return nil, fmt.Errorf("unexpected value " +
					"for '--%s': '%s'", optname, optval)
			}
		}

		if longs[i].value {
			return func(val string) error {
				var err error = longs[i].cb(val)

				if err == nil {
					return nil
				}

				return fmt.Errorf("invalid value for " +
						"'--%s': %s", optname,
					err.Error())
			}, nil
		}

		err = longs[i].cb(optval)

		if err != nil {
			return nil, fmt.Errorf("invalid value for '--%s': %s",
				optname, err.Error())
		}

		return nil, nil
	}

	return nil, fmt.Errorf("unknown option '%s'", arg)
}

func parseShortOptions(arg string, shorts []shortOption) (func(string) error, error) {
	var found bool
	var err error
	var i, j int

	for i = range arg[1:] {
		found = false

		for j = range shorts {
			if arg[i+1] != shorts[j].name {
				continue
			}

			if shorts[j].value {
				if (i+2) == len(arg) {
					return func(val string) error {
						var err error =
							shorts[j].cb(val)

						if err == nil {
							return nil
						}

						return fmt.Errorf("invalid " +
							"value for '%c' in " +
							"'%s': %s", arg[i+1],
							arg, err.Error())
					}, nil
				}

				err = shorts[j].cb(arg[i+2:])

				if err != nil {
					return nil, fmt.Errorf("invalid " +
						"value for '%c' in '%s': %s",
						arg[i+1], arg, err.Error())
				}

				return nil, nil
			}

			err = shorts[j].cb("")

			if err != nil {
				return nil, fmt.Errorf("invalid value for " +
					"'%c' in '%s': %s", arg[i+1], arg,
					err.Error())
			}

			found = true
			break
		}

		if !found {
			return nil, fmt.Errorf("unknown option '%c' in '%s'",
				arg[i+1], arg)
		}
	}

	return nil, nil
}

func parseOptions(args []string, shorts []shortOption, longs []longOption) (int, error) {
	var wantVal func(string) error
	var err error
	var i int

	for i = 0; i < len(args); i++ {
		if wantVal != nil {
			err = wantVal(args[i])
			if err != nil {
				return i, err
			}

			wantVal = nil
			continue
		}

		if strings.HasPrefix(args[i], "--") {
			if len(args[i]) == 2 {
				break
			}

			wantVal, err = parseLongOption(args[i], longs)
			if err != nil {
				return i, err
			}

			continue
		}

		if strings.HasPrefix(args[i], "-") {
			wantVal, err = parseShortOptions(args[i], shorts)
			if err != nil {
				return i, err
			}

			continue
		}

		break
	}

	if wantVal != nil {
		return i, fmt.Errorf("missing value")
	}

	return i, nil
}
