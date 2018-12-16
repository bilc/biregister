biregister是基于etcd v3版本实现的服务注册发现library，使用golang实现。  
包含服务注册和服务发现两个接口。 

# 概念  

prefix：注册的节点前缀　  
name：注册名称。etcd key中，除去prefix的部分   
value：为注册节点的值  
ttl：与etcd连接超时时间  　　

# Watcher  

- 服务发现接口  
用于观察一个prefix开头的所有节点的变化,并对这些节点和值做了缓存，以便能够读取缓存或者选举出leader。 
```
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
```
- 初始化方法  
```
func NewWatcher(etcdServs []string, prefix string, ttl int64) (w *watcher, err error) 
```

# Register  

- 服务注册接口（支持服务发现接口所有方法）  
将自己注册到一个prefix开头的节点上，并保持心跳连接。  
```
type Register interface {
	MyKey() string
	MyName() string
	AmILeader() bool
	Watcher
}
```
- 初始化方法  
有两个函数：不提供name和提供name的函数； 不提供name时，会使用当前Revision做为name
```
func NewRegisterNoName(etcdServs []string, prefix string, value string, ttl int64) (*register, error) 
func NewRegister(etcdServs []string, prefix string, name, value string, ttl int64) (*register, error) 
```
