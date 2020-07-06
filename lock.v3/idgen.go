package lock

import (
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"os"
	"sync"
	"time"
)

var pid = uint16(time.Now().UnixNano() & 65535)
var machineFlag uint16
var onece = sync.Once{}

func getMachineFlag() uint16 {
	onece.Do(func() {
		hostname, _ := os.Hostname()
		machineFlag = hashNum(hostname)
	})
	return machineFlag
}

func hashNum(str string) uint16 {
	tempv := int(str[0])
	for _, ruv := range str {
		tempv = 40503*tempv + int(ruv)
	}
	tempv &= 65535
	return uint16(tempv)
}

func idGen() string {
	var b [16]byte
	binary.LittleEndian.PutUint16(b[:], pid)
	binary.LittleEndian.PutUint16(b[2:], getMachineFlag())
	binary.LittleEndian.PutUint64(b[4:], uint64(time.Now().UnixNano()))
	binary.LittleEndian.PutUint32(b[12:], rand.Uint32())
	return base64.URLEncoding.EncodeToString(b[:])
}
