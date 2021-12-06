package cache

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	Get(cacheGroup string, key string) ([]byte, error)
}
