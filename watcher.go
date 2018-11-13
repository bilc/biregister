/**********************************************************
 * Author        : blc
 * Last modified : 2018-11-12 16:43
 * Filename      : watcher.go
 * Description   : leader election and cache For some etcd path(prefix)
 * *******************************************************/
package biregister

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

type Change struct {
	Name string
	Op   string //"PUT" "DELETE"
}

/*
* name: key without path
* value:
 */
type Watcher interface {
	//cache interface
	GetAll() map[string]string
	GetNames() []string
	GetValues() []string
	GetValueByName(name string) string

	//leader election interface
	GetLeader() (name, val string)

	Close()
	Changes() <-chan Change
	Closed() <-chan struct{}

	EtcdClient() *clientv3.Client
}

type watcher struct {
	etcdServers []string
	ttl         int64
	prefix      string

	etcdCli   *clientv3.Client
	eventChan clientv3.WatchChan

	all                  map[string]content
	masterName           string
	masterCreateRevision int64
	cacheMutex           sync.Mutex

	closeChan  chan struct{}
	changeChan chan Change
}

type content struct {
	val            string
	createRevision int64
}

var DefaultKeepAlive time.Duration = 2 * time.Second
var DefaultTimeout time.Duration = 5 * time.Second
var DefaultChanSize int = 100

func NewWatcher(etcdServs []string, prefix string, ttl int64) (w *watcher, err error) {

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            etcdServs,
		DialTimeout:          DefaultTimeout,
		DialKeepAliveTime:    DefaultKeepAlive,
		DialKeepAliveTimeout: DefaultTimeout,
	})
	if err != nil {
		return nil, err
	}
	w = &watcher{etcdServers: etcdServs,
		ttl:                  ttl,
		prefix:               prefix,
		etcdCli:              cli,
		all:                  make(map[string]content),
		masterName:           "",
		masterCreateRevision: math.MaxInt64,
		closeChan:            make(chan struct{}, DefaultChanSize),
		changeChan:           make(chan Change, DefaultChanSize),
	}
	return w, w.updateLoop()
}

func (w *watcher) EtcdClient() *clientv3.Client {
	return w.etcdCli
}

func (w *watcher) Closed() <-chan struct{} {
	return w.closeChan
}

func (w *watcher) Close() {
	w.etcdCli.Close()
	close(w.closeChan)
}

func (w *watcher) Changes() <-chan Change {
	return w.changeChan
}

func (w *watcher) GetNames() []string {
	ret := make([]string, 0, len(w.all))
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	for k, _ := range w.all {
		ret = append(ret, k)
	}
	return ret
}

func (w *watcher) GetValues() []string {
	ret := make([]string, 0, len(w.all))
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	for _, v := range w.all {
		ret = append(ret, v.val)
	}
	return ret
}

func (w *watcher) GetAll() map[string]string {
	ret := make(map[string]string)
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	for k, v := range w.all {
		ret[k] = v.val
	}
	return ret
}

func (w *watcher) GetValueByName(name string) string {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	if v, ok := w.all[name]; ok {
		return v.val
	} else {
		return ""
	}
}

func (w *watcher) GetLeader() (string, string) {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()
	if w.masterName != "" {
		return w.masterName, w.all[w.masterName].val
	} else {
		return "", ""
	}
}

func (w *watcher) updateLoop() error {
	resp, err := w.update()
	if err != nil {
		close(w.closeChan)
		return err
	}

	go func() {
		w.eventChan = w.etcdCli.Watch(context.Background(),
			w.prefix,
			clientv3.WithPrefix(),
			clientv3.WithRev(resp.Header.Revision+1)) // do not miss any change

		for {
			wcRsp, ok := <-w.eventChan
			if ok {
				w.cacheMutex.Lock() // lock .....
				for _, e := range wcRsp.Events {
					name := string(e.Kv.Key[len(w.prefix):])
					switch e.Type {
					case mvccpb.PUT:
						w.all[name] = content{string(e.Kv.Value), e.Kv.CreateRevision}
					case mvccpb.DELETE:
						delete(w.all, name)
						if name == w.masterName {
							w.masterName = ""
							w.masterCreateRevision = math.MaxInt64
						}
					}
					w.changeChan <- Change{name, e.Type.String()}
				}

				if w.masterName == "" {
					for k, v := range w.all {
						if v.createRevision < w.masterCreateRevision {
							w.masterName = k
							w.masterCreateRevision = v.createRevision
						}
					}
				}
				w.cacheMutex.Unlock() // unlock .....
			} else {
				return
			}
		}
	}()
	return nil
}

func (w *watcher) update() (resp *clientv3.GetResponse, err error) {
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
	w.cacheMutex.Lock() // lock .....
	if len(resp.Kvs) != 0 {
		w.masterCreateRevision = resp.Kvs[0].CreateRevision
		w.masterName = string(resp.Kvs[0].Key[len(w.prefix):])
	}
	for _, j := range resp.Kvs {
		w.all[string(j.Key[len(w.prefix):])] = content{string(j.Value), j.CreateRevision}
	}
	w.cacheMutex.Unlock()
	return resp, nil
}
