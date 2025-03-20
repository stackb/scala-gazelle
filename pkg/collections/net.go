package collections

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const connectionAvailableDuration = 250 * time.Millisecond

// WaitForConnectionAvailable pings a tcp connection every 250 milliseconds
// until it connects and returns true.  If it fails to connect by the timeout
// deadline, returns false.
func WaitForConnectionAvailable(host string, port int, timeout time.Duration, progress bool) bool {
	target := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	var wg sync.WaitGroup
	wg.Add(1)

	success := make(chan bool, 1)

	go func() {
		go func() {
			defer wg.Done()
			for {
				_, err := net.Dial("tcp", target)
				if err == nil {
					break
				}
				if progress {
					log.Println(err)
				}
				time.Sleep(connectionAvailableDuration)
			}
		}()
		wg.Wait()
		success <- true
	}()

	select {
	case <-success:
		return true
	case <-time.After(timeout):
		return false
	}
}
