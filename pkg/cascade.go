package pigo

type Cascade interface {
	Unpack(cascade []byte) *PuplocCascade
}
