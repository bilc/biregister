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
	r, err := NewRegister([]string{"http://127.0.0.1:2379"}, "/asdf", "111.222.333.444:555", "111.222.333.444:555-v", 2)
	go func() {
		for {
			c := <-r.Changes()
			fmt.Println("changes", c)
		}
	}()
	assert.Nil(t, err)
	time.Sleep(time.Second)

	masterName, masterValue := r.GetLeader()
	fmt.Println("testOne: ", masterName, masterValue, r.GetNames())
	assert.Equal(t, r.myValue, masterValue)
	m := r.AmILeader()
	assert.True(t, m)
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
	r1, err := NewRegister([]string{url}, prefix, name1, v1, 2)
	assert.Nil(t, err)

	name2 := "2222"
	v2 := "2.2.3.4:1234"
	r2, err := NewRegister([]string{url}, prefix, name2, v2, 2)
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
	assert.True(t, r1.AmILeader())
	assert.False(t, r2.AmILeader())

	assert.Subset(t, []string{name1, name2}, r1.GetNames())
	assert.Subset(t, []string{v1, v2}, r1.GetValues())

	r1.Close()
	time.Sleep(time.Second * 3)
	assert.Equal(t, []string{v2}, r2.GetValues())

}
