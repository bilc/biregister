package biregister

import (
	//	"fmt"
	"time"
	//	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatcherBase(t *testing.T) {
	//return
	url := "http://127.0.0.1:2379"
	//cmd := exec.Command("etcd", "--advertise-client-urls", url, "--listen-client-urls", url)
	//cmd.Start()

	prefix := "/watcherbase"
	name1 := "1111"
	v1 := "1.2.3.4:1234"
	r1, err := NewRegisterWithName([]string{url}, prefix, name1, v1, 2)
	assert.Nil(t, err)

	name2 := "2222"
	v2 := "2.2.3.4:1234"
	r2, err := NewRegisterWithName([]string{url}, prefix, name2, v2, 2)
	assert.Nil(t, err)
	assert.NotNil(t, r2)

	<-r1.Changes()
	assert.Equal(t, v1, r1.GetValueByName(name1))
	<-r1.Changes()
	assert.Equal(t, v2, r1.GetValueByName(name2))
}
