package biregister

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOneRegistry(t *testing.T) {
	cmd := exec.Command("etcd")
	cmd.Start()
	time.Sleep(time.Second)
	r, err := NewRegister([]string{"http://127.0.0.1:2379"}, "/asdf", "111.222.333.444:555", 2)
	assert.Nil(t, err)
	time.Sleep(time.Second)
	m := r.AmIMaster()
	fmt.Println(r.myKey, r.GetMasterKey())
	assert.True(t, m)
	master := r.GetMasterValue()
	assert.Equal(t, r.myValue, master)
}

func TestTwoRegistry(t *testing.T) {
	url := "http://127.0.0.1:2379"
	cmd := exec.Command("etcd", "--advertise-client-urls", url, "--listen-client-urls", url)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	cmd.Start()

	prefix := "axdfxxx"
	name1 := "1111"
	v1 := "1.2.3.4:1234"
	r1, err := NewRegisterWithName([]string{url}, prefix, name1, v1, 2)
	assert.Nil(t, err)

	name2 := "2222"
	v2 := "2.2.3.4:1234"
	r2, err := NewRegisterWithName([]string{url}, prefix, name2, v2, 2)
	assert.Nil(t, err)
	go func() {
		for {
			select {
			case <-r2.Changes():
				fmt.Println("r2 ", r2.GetValues())
			}
		}
	}()

	time.Sleep(time.Second)
	assert.True(t, r1.AmIMaster())
	assert.False(t, r2.AmIMaster())

	assert.Equal(t, []string{name1, name2}, r1.GetNames())
	assert.Equal(t, []string{v1, v2}, r1.GetValues())

	r1.Stop()
	time.Sleep(time.Second * 3)
	assert.Equal(t, []string{v2}, r2.GetValues())

	//begin1 := time.Now()
	//r1.WaitBeMaster()
	//end1 := time.Now()
	//assert.True(t, end1.Sub(begin1) < time.Second)

	//r1.Stop()
	//begin := time.Now().Second()
	//r2.WaitBeMaster()
	//end := time.Now().Second()
	//fmt.Println("waiting time", end-begin)

	//r3, err := NewRegister([]string{url}, dir, "888.222.333.444:555", 2)
	//time.Sleep(time.Second)
	//assert.Nil(t, err)
	//assert.Equal(t, r3.GetMasterValue(), "999.222.333.444:555")
}
