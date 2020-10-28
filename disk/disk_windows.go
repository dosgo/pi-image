package disk

import (
	"fmt"
	"golang.org/x/sys/windows"
	"log"
	"pi-image/comm"
	"strings"
)
import "github.com/StackExchange/wmi"



type storageInfo struct {
	DeviceID       string
	PNPDeviceID       string
	Manufacturer  string
	Model string
	SerialNumber string
	Name string
}


type logicalDisk struct {
	DeviceID       string
	Name string
	VolumeName string
	PNPDeviceID string
}

type Win32_DiskPartition struct{
	Name string
}


type Win32_DiskDriveToDiskPartition  struct {
	//Win32_DiskDrive      storageInfo
	//Win32_DiskPartition  Win32_DiskPartition
	//Antecedent string;
	//VolumeName string;
	DeviceID string;
	Description string;
	Name string;
}



func GetStorageInfo(usb bool) []comm.DiskInfo {
	var storageinfo []storageInfo
	var logicalDisk []Win32_DiskDriveToDiskPartition
	var diskPart []Win32_DiskDriveToDiskPartition
	var where="";
	var disks = make([]comm.DiskInfo, 0)
	if(usb) {
		where = " where InterfaceType='USB'";
	}
	err := wmi.Query("SELECT * FROM Win32_DiskDrive"+where, &storageinfo)
	if err != nil {
		return disks;
	}
	for _, storage := range storageinfo {
		err1 := wmi.Query("ASSOCIATORS OF {Win32_DiskDrive.DeviceID='"+ storage.DeviceID +"'}  WHERE AssocClass = Win32_DiskDriveToDiskPartition ", &diskPart)
		if(err1==nil){
			var _path="";
			var _name=storage.Model;
			var _allPath="";
			for _, storage1 := range diskPart {
				err = wmi.Query("ASSOCIATORS OF {Win32_DiskPartition.DeviceID='"+storage1.DeviceID+"'} WHERE AssocClass = Win32_LogicalDiskToPartition",&logicalDisk);
				if(err1==nil){
					for _, logicalD := range logicalDisk {
						if(_path==""){
							_path=strings.TrimSpace(logicalD.Name)
						}
						_allPath=_allPath+strings.TrimSpace(logicalD.Name)
					}
				}
			}
			_disk := comm.DiskInfo{
				Path:_path[:1],
				Name: _name+"("+_allPath+")",
			}
			disks = append(disks, _disk)
		}else{
			fmt.Printf("err1:%v\r\n",err1)
		}
	}
	return disks;
}

func ReadDiskBuf(dev string,_len int) ([]byte,error){
	//通过CreateFile来获得设备的句柄
	hDevice,err:= windows.CreateFile(windows.StringToUTF16Ptr(dev), // 设备名称,这里指第一块硬盘
		windows.GENERIC_READ,                // no access to the drive
		windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE,  // share mode
		nil,             // default security attributes
		windows.OPEN_EXISTING,    // disposition
		0,                // file attributes
		0);            // do not copy file attributes
	if err != nil{
		log.Printf("Creatfile error!May be no permission!ERROR_ACCESS_DENIED！\n");
		return nil,err;
	}
	//读取MBR
	var  MbrBuf =make([]byte,_len);
	var len uint32=uint32(_len);
	err=windows.ReadFile(hDevice, MbrBuf, &len,nil);
	if(err!=nil){
		return nil,err;
	}
	return MbrBuf,nil;
}

