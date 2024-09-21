package collections

import (
	"fmt"
	"os"
)

func PrintProcessIdForDelveAndWait() {
	fmt.Printf("Debugging session requested (Process ID: %d)\n", os.Getpid())
	fmt.Printf("dlv attach --headless --listen=:2345 %d\n", os.Getpid())
	fmt.Println("Press ENTER to continue.")
	fmt.Scanln()
}
