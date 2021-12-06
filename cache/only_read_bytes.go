package cache

// OnlyReadBytes 存储真实的缓存值 缓存值的抽象与封装
// byte类型可以支持任意的数据类型的存储，例如字符串、图片。
type OnlyReadBytes struct {
	b []byte
}

// Len 返回所占内存大小
func (orb OnlyReadBytes) Len() int {
	return len(orb.b)
}

// ByteSlice 返回切片,防止内存被外部篡改
func (orb OnlyReadBytes) ByteSlice() []byte {
	return cloneBytes(orb.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// String 返回字符串类型
func (orb OnlyReadBytes) String() string {
	return string(orb.b)
}
