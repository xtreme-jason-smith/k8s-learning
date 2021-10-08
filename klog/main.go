package main

import (
	"flag"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
)

//This example file was based off of https://github.com/kubernetes/klog/blob/main/examples/klogr/main.go

type myError struct {
	str string
}

func (e myError) Error() string {
	return e.str
}

func main() {
	//Initialize klog.
	klog.InitFlags(nil)
	_ = flag.Set("v", "3") // Sets the Info verbosity level for klog
	flag.Parse()

	// Setup information that will be outputted in all logs
	log := klogr.New().WithName("TestName").WithValues("pod", "192.168.0.1")

	// A basic info message. This will always show up regardless of the verbosity level
	log.Info("Hello,", "World")

	// Creating another log that will output "container"="4" key value pair to all logs that use log2
	// This will always show up regardless of the verbosity level
	log2 := log.WithValues("container", "4")

	// Example log including a structure
	// This will always show up regardless of the verbosity level
	log.Info("hello", "val1", 1, "val2", map[string]int{"k": 1})

	// Example Debug log: Verbosity level 4 is considered debug log. This will not show up in the log output as klog was configured to only output log levels 3 and lower.
	log.V(4).Info("nice to meet you")
	// This does the same thing as the line above
	if log.V(4).Enabled() {
		log.Info("nice to meet you too")
	}
	
	// Example error logs
	log.Error(nil, "uh oh", "trouble", true, "reasons", []float64{0.1, 0.11, 3.14})
	log.Error(myError{"an error occurred"}, "goodbye", "code", -1)
	
	// Example log that will include "Testname", "pod" key value, and "container" key value in all logs (it was created on line 33)
	// These logs will only show up if verbosity level is set to 3 or higher
	log2.V(3).Info("New level log")
	log2.V(3).Info("Testing with keys values", "some key", "some value", "anotherkey", "anothervalue")
	klog.Flush()
}
