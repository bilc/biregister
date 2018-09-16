package biregister

import (
	"context"
	"fmt"

	"github.com/coreos/etcd/clientv3"
)

type Register struct {
	*Watcher

	myName  string
	myKey   string
	myValue string
}

func NewRegister(etcdServs []string, prefix string, value string, ttl int64) (*Register, error) {
	return NewRegisterWithName(etcdServs, prefix, "", value, ttl)
}

func NewRegisterWithName(etcdServs []string, prefix string, name, value string, ttl int64) (*Register, error) {
	w, err := NewWatcher(etcdServs, prefix, ttl)
	if err != nil {
		return nil, err
	}
	r := &Register{Watcher: w, myName: name, myValue: value}
	if err := r.registerMyself(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Register) MyKey() string {
	return r.myKey
}

func (r *Register) MyName() string {
	return r.myName
}

func (r *Register) AmIMaster() bool {
	return r.myKey == r.GetMasterKey()
}

func (r *Register) registerMyself() error {
	// create lease
	var leaseID clientv3.LeaseID
	resp, err := r.etcdCli.Grant(context.Background(), r.ttl)
	if err != nil {
		return err
	}
	// keep alive
	leaseID = resp.ID
	ch, err := r.etcdCli.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return err
	}

	// check alive status
	go func() {
		for {
			_, ok := <-ch
			if !ok {
				return
			}
		}
	}()

	// put key(newest reversion) to etcd
	count := 0
	for count = 0; count < 10; count++ {
		key := r.prefix + r.myName
		if r.myName == "" {
			resp, err := r.etcdCli.Get(context.Background(), "/")
			if err != nil {
				return err
			}
			key = fmt.Sprintf("%s/%020d", r.prefix, resp.Header.Revision+1)
		}

		tresp, err := r.etcdCli.Txn(context.Background()).
			If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
			Then(clientv3.OpPut(key, r.myValue, clientv3.WithLease(leaseID))).
			Else().
			Commit()
		if err != nil {
			return err
		}
		if tresp.Succeeded {
			r.myKey = key
			r.myName = key[len(r.prefix):]
			break
		}
	}
	if count == 10 {
		return fmt.Errorf("register %v error: retry times > 10 ", r.prefix+r.myName)
	}
	return nil
}
