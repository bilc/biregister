package biregister

import (
	"fmt"
	//	"os/exec"
	"testing"
	"time"

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
	r1, err := NewRegister([]string{url}, prefix, name1, v1, 2)
	assert.Nil(t, err)

	name2 := "2222"
	v2 := "2.2.3.4:1234"
	r2, err := NewRegister([]string{url}, prefix, name2, v2, 2)
	assert.Nil(t, err)
	assert.NotNil(t, r2)

	time.Sleep(time.Second)
	<-r1.Changes()
	assert.Equal(t, v1, r1.GetValueByName(name1))
	<-r1.Changes()
	assert.Equal(t, v2, r1.GetValueByName(name2))

	fmt.Println(r1.GetNames())
	assert.Equal(t, []string{name1, name2}, r1.GetNames())
}
