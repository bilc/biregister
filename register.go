/**********************************************************
 * Author        : blc
 * Last modified : 2018-11-13 09:55
 * Filename      : register.go
 * Description   : register For living report
 * *******************************************************/
package biregister

import (
	"context"
	"fmt"

	"github.com/coreos/etcd/clientv3"
)

type Register interface {
	MyKey() string
	MyName() string
	AmILeader() bool
	Watcher
}

type register struct {
	*watcher

	myName  string
	myKey   string
	myValue string
}

func NewRegisterNoName(etcdServs []string, prefix string, value string, ttl int64) (*register, error) {
	return NewRegister(etcdServs, prefix, "", value, ttl)
}

func NewRegister(etcdServs []string, prefix string, name, value string, ttl int64) (*register, error) {
	w, err := NewWatcher(etcdServs, prefix, ttl)
	if err != nil {
		return nil, err
	}
	r := &register{watcher: w, myName: name, myValue: value}
	if err := r.registerMyself(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *register) MyKey() string {
	return r.myKey
}

func (r *register) MyName() string {
	return r.myName
}

func (r *register) AmILeader() bool {
	name, _ := r.GetLeader()
	return r.myName == name
}

func (r *register) registerMyself() error {
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
		return nil
	} else {
		return fmt.Errorf("resp %v", tresp)
	}
}
