# Understanding Logging

In the Diablo project, we use the `zap` logger provided by Uber.

You can find the project home: [ http://github.com/uber-go/zap ](http://github.com/uber-go/zap)
And relevant documentation: [ https://pkg.go.dev/go.uber.org/zap ](https://pkg.go.dev/go.uber.org/zap)

## Important information

Currently, the logger is configured as a "teed logger", which provides both
`stdout` as well as logging to the related files.

`primary_diablo.log` and `secondary_diablo.log` are the allocated logs for the primary/secondary on the machine.


## How to log?

To integrate the log into your client, you will need to import the zap library.
Once done, it will take the global configuration, so you just need to call the log level.

Example:

```go
import "go.uber.org/zap"

x := 2

zap.L().Info("Hello from INFO level log",
	zap.Int("this is an int", x))
```

As Zap is a strongly typed logger that allows for concurrent access, you will
need to specifically state the type. All information can be found in their [docs](https://pkg.go.dev/go.uber.org/zap#Field)
