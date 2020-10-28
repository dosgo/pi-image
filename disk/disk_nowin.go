// +build !windows

package disk

import (
	"os"
)

func ReadDiskBuf(dev string,_len int) ([]byte,error){
	f, _ := os.Open(dev)
	defer f.Close();
	buf := make([]byte, _len)
	_,err:=f.Read(buf)
	if(err!=nil){
		return nil,err;
	}
	return buf,nil;
}
