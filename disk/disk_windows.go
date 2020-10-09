package disk

import (
	"fmt"
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
			for _, storage1 := range diskPart {
				err = wmi.Query("ASSOCIATORS OF {Win32_DiskPartition.DeviceID='"+storage1.DeviceID+"'} WHERE AssocClass = Win32_LogicalDiskToPartition",&logicalDisk);
				if(err1==nil){
					for _, logicalD := range logicalDisk {
						_path:=strings.TrimSpace(logicalD.Name)
						_disk := comm.DiskInfo{
							Path:_path[:1],
							Name: storage.Model,
						}
						disks = append(disks, _disk)
					}
				}
			}
		}else{
			fmt.Printf("err1:%v\r\n",err1)
		}
	}
	return disks;
}

