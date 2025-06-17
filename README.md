# goth

Simplified wrapper for goroutines that provides:  

✔ **Lifecycle control** — start, stop, and state monitoring  
✔ **Auto-recovery** — automatic restart of failed tasks  
✔ **Prometheus integration** — built-in execution metrics  
✔ **Minimalist API** — easy integration with existing projects  

Ideal for background jobs, event handlers, long-running processes.

## Installation

```bash
go get github.com/h0st1s/goth
```



## Usage

Import a package:
```go
import (
    "github.com/h0st1s/goth"
)
```



Task example:
```go
func example(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            fmt.Printf("> do some stuff...\n")
            time.Sleep(time.Second)
        }
    }
}
```


Configuration:
```go
conf := goth.ThreadConf{
    Restart:      -1,
    RestartDelay: time.Second,
}
```
`Restart` meanings:
- `-1` -- infinite restart
- `0` -- execute once
- `n` -- execute and then restart n-times


Run:
```go
worker := goth.NewThread(conf, example).Run()
defer worker.Terminate() // stop and cleanup 
```


Run with context:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

worker := goth.NewThread(conf, example).
    WithContext(ctx).
    Run()
defer worker.Terminate() // stop and cleanup 
```


Run with prometheus collector:
```go
worker := goth.NewThread(conf, example).
    WithCollector("worker").
    Run()
defer worker.Terminate() // stop and cleanup 
```


Function `WithCollector()` will create and register stat collector:
```
# HELP goth_threads_crashes The number of the thread crashes
# TYPE goth_threads_crashes counter
goth_threads_crashes{thread="worker"} 0
# HELP goth_threads_restarts The number of the thread restarts
# TYPE goth_threads_restarts counter
goth_threads_restarts{thread="worker"} 2
# HELP goth_threads_status Thread status
# TYPE goth_threads_status gauge
goth_threads_status{status="crashed",thread="worker"} 0
goth_threads_status{status="running",thread="worker"} 1
goth_threads_status{status="stopped",thread="worker"} 0
goth_threads_status{status="waiting",thread="worker"} 0
```


Start/Stop example:
```go
worker := goth.NewThread(conf, example).Run()

// do something here
// ...

<-worker.Stop() // signal to stop and wait until worker stops

// do something here too
// ...

worker.Run() // run worker again

// do something again
// ...

worker.Terminate() // stop worker and cleanup
```

See more code examples in [examples](/examples)


## Documentation

TODO


## License

[MIT](LICENSE)


