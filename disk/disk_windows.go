package disk

import (
	"fmt"
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




func GetStorageInfo() {
	var storageinfo []storageInfo
	var logicalDisk []Win32_DiskDriveToDiskPartition
	var diskPart []Win32_DiskDriveToDiskPartition
	err := wmi.Query("SELECT * FROM Win32_DiskDrive", &storageinfo)
	if err != nil {
		return
	}
	for _, storage := range storageinfo {
		fmt.Printf("dd:%s name:%s %s %s \r\n",storage.DeviceID,storage.Name,storage.Model,storage.SerialNumber);

		err1 := wmi.Query("ASSOCIATORS OF {Win32_DiskDrive.DeviceID='"+ storage.DeviceID +"'}  WHERE AssocClass = Win32_DiskDriveToDiskPartition ", &diskPart)
		if(err1==nil){
			for _, storage1 := range diskPart {

				err = wmi.Query("ASSOCIATORS OF {Win32_DiskPartition.DeviceID='"+storage1.DeviceID+"'} WHERE AssocClass = Win32_LogicalDiskToPartition",&logicalDisk);
				if(err1==nil){
					for _, logicalD := range logicalDisk {
						fmt.Printf("logicalDisk:%s  %s \r\n",logicalD.Name,logicalD.DeviceID)
					}

				}
				fmt.Printf("storage1:%v \r\n",storage1)
			}

		}else{
			fmt.Printf("err1:%v\r\n",err1)
		}
	}



}

