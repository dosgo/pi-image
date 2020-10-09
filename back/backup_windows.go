package back

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"pi-image/comm"
	"pi-image/disk"
	"pi-image/winDump"
	"unsafe"
)
/*
* 本机备份需要
selfBack true
*/

func Backup(devName string, imgFile string, selfBack bool) error {
	errCode:=winDump.DumpImg(devName,imgFile);
	if(errCode==0){
		return nil;
	}
	return errors.New("back err！")
}


func GetDisk() []comm.DiskInfo {
	return disk.GetStorageInfo(true);
}

func GetSelfDev(bootPath string) string {
	return ""
}

func GetSelfBootDev() string {
	return ""
}

func CheckPm(){
	if !IsRunasAdmin()  {
		fmt.Printf("Please run with administrator.\r\n")
		os.Exit(-1)
	}
}

func IsRunasAdmin() bool{
	var hToken windows.Token
	p := windows.CurrentProcess()
	// Get current process token
	err:=windows.OpenProcessToken(p, windows.TOKEN_QUERY, &hToken )
	if(err!=nil){
		return false;
	}

	var tokenEle uint32
	var dwRetLen uint32

	// Retrieve token elevation information
	err = windows.GetTokenInformation( hToken,windows.TokenElevation, (*byte)(unsafe.Pointer(&tokenEle)), uint32(unsafe.Sizeof(tokenEle)), &dwRetLen )
	if err != nil {
		return false
	}
	return dwRetLen == uint32(unsafe.Sizeof(tokenEle)) && tokenEle != 0
}
