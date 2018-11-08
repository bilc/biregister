package biregister

import (
	"context"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type Watcher struct {
	etcdServers []string
	ttl         int64
	prefix      string

	etcdCli *clientv3.Client

	values     []string
	keys       []string
	cacheMutex sync.Mutex

	closeChan  chan struct{}
	changeChan chan struct{}
}

var DefaultKeepAlive time.Duration = 2 * time.Second
var DefaultTimeout time.Duration = 5 * time.Second
var DefaultChanSize int = 100

func NewWatcher(etcdServs []string, prefix string, ttl int64) (w *Watcher, err error) {

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            etcdServs,
		DialTimeout:          DefaultTimeout,
		DialKeepAliveTime:    DefaultKeepAlive,
		DialKeepAliveTimeout: DefaultTimeout,
	})
	if err != nil {
		return nil, err
	}
	w = &Watcher{etcdServers: etcdServs,
		ttl:        ttl,
		prefix:     prefix,
		etcdCli:    cli,
		values:     make([]string, 0),
		keys:       make([]string, 0),
		closeChan:  make(chan struct{}, DefaultChanSize),
		changeChan: make(chan struct{}, DefaultChanSize),
	}
	//	_, err = w.update()
	//	if err != nil {
	//		return nil, err
	//	}
	go w.updateLoop()
	return w, nil
}

func (w *Watcher) EtcdClient() *clientv3.Client {
	return w.etcdCli
}

func (w *Watcher) Stoped() <-chan struct{} {
	return w.closeChan
}

func (w *Watcher) Stop() {
	w.etcdCli.Close()
	close(w.closeChan)
}

func (w *Watcher) Changes() <-chan struct{} {
	return w.changeChan
}

func (w *Watcher) GetValues() []string {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	ret := make([]string, 0, len(w.values))
	for _, j := range w.values {
		ret = append(ret, j)
	}
	return ret
}

func (w *Watcher) GetValueByName(name string) string {
	names := w.GetNames()
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	for i, n := range names {
		if n == name {
			return w.values[i]
		}
	}
	return ""
}

func (w *Watcher) GetMasterValue() string {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	if len(w.values) != 0 {
		return w.values[0]
	} else {
		return ""
	}
}

func (w *Watcher) GetMasterKey() string {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	if len(w.keys) != 0 {
		return w.keys[0]
	} else {
		return ""
	}
}

func (w *Watcher) GetNames() []string {
	ret := make([]string, 0, len(w.keys))
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	for _, key := range w.keys {
		ret = append(ret, key[len(w.prefix):])
	}
	return ret
}

func (w *Watcher) update() (resp *clientv3.GetResponse, err error) {
	// get all member
	begin := time.Now()
	for {
		// get all sorted keys
		resp, err = w.etcdCli.Get(context.Background(), w.prefix, clientv3.WithPrefix(),
			clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortAscend))
		if err == nil {
			break
		}
		time.Sleep(time.Second)
		if time.Now().Sub(begin) > time.Duration(w.ttl)*time.Second {
			return nil, err
		}
	}

	// update all data
	w.cacheMutex.Lock()     // lock .....
	w.values = w.values[:0] // clean up
	w.keys = w.keys[:0]
	for _, j := range resp.Kvs {
		w.values = append(w.values, string(j.Value))
		w.keys = append(w.keys, string(j.Key))
	}
	w.cacheMutex.Unlock()
	return resp, nil
}

func (w *Watcher) updateLoop() {
	for {
		resp, err := w.update()
		if err != nil {
			close(w.closeChan)
			return
		}
		//fmt.Printf("!!!etcdvalues:%p,%v \n", w, resp)

		var tmp struct{}
		select {
		case w.changeChan <- tmp:
		default:
		}

		// wait next change
		watcher := clientv3.NewWatcher(w.etcdCli)
		wc := watcher.Watch(context.Background(),
			w.prefix,
			clientv3.WithPrefix(),
			clientv3.WithRev(resp.Header.Revision+1)) // do not miss any change
		select {
		case <-wc:
		case <-w.closeChan:
			return
		}
		watcher.Close()
	}
}
