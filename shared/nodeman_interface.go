package shared

type NodemanIF interface {
    GetKey(string) ([]byte, error)
}
