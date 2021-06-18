# retry

A simple retry mechanism for Go.

Given a function that returns an error, `Retry()` will execute the function
repeatedly until the function returns without error or the loop is terminated.
Loop termination is controlled by a `Limiter` function that is called after
each failed attempt. If the limiter allows the loop to continue a `Timer`
function is called and the subsequent attempt is not made until the timer
returns.

Limiters and timers are easy to write but this package provides a number of
useful implementations out of the box.

## Example

The following example illustrates a command-line program. It attempts the `work()`
function, which unfortunately cannot ever succeed. Before it calls into the retry
loop it sets up a signal handler to catch a keyboard interrupt (ctrl+c). When the
interrupt is received it causes the cancelation of a `context.Context`. The retry
loop will cycle forever (because the `work()` function will never succeed) with
a one-second sleep between each invocation until the cancelation is received.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/colvin/retry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)

	go func() {
		<-signals
		cancel()
	}()

	err := retry.Retry(work, retry.UntilCanceled(ctx), retry.CancelableSleep(ctx, time.Duration(1 * time.Second)))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("success")
}

func work() error {
	fmt.Println("working...")
	return fmt.Errorf("oops, all failure")
}
```
