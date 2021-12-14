package server

import (
	"fmt"
	"testing"
)

func Test_NewController(t *testing.T) {
	ser := NewCloudEventServer()
	err := ser.Run(fmt.Sprintf(":9081"))
	if err != nil {
		t.Fatal(err)
	}
}
