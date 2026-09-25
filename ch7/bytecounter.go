// Package ch7 provides a ByteCounter type that implements the io.Writer interface.
package ch7

import "fmt"

type ByteCounter int

func (c *ByteCounter) Write(p []byte) (int, error) {

	*c += ByteCounter(len(p))
	return len(p), nil

}

func TestByteCounter() {

	var c ByteCounter
	c.Write([]byte("hello"))
	fmt.Println(c) // "5", len("hello")

}
