package main

import (
	"fmt"
	"pi-image/back"
	"time"

	_ "github.com/diskfs/go-diskfs"
	_ "github.com/diskfs/go-diskfs/disk"
	_ "github.com/diskfs/go-diskfs/partition"
)
var version="v1.0-(2020-10-05)"

//https://github.com/dsoprea/go-ext4
//https://github.com/nerd2/gexto
//https://github.com/paulmey/inspect-azure-vhd/
//https://github.com/diskfs/go-diskfs
func main() {
	fmt.Printf("pi-image V:%s\r\n",version)
	back.CheckCmd()
	dialog()
}

func dialog() {
	var diskNum = -1
	disks := back.GetDisk()
	selfDev := back.GetSelfDev(back.GetSelfBootDev())
	for {
		if diskNum == -1 {
			fmt.Println("Please select a backup disk.")
		} else {
			fmt.Println("Please re-select backup disk.")
		}
		for _k, v := range disks {
			if selfDev == v.Path {
				fmt.Println(fmt.Sprintf("%d.%s(Current system)", _k, v.Path))
			} else {
				fmt.Println(fmt.Sprintf("%d.%s(%s)", _k, v.Path, v.Name))
			}
		}
		fmt.Scanf("%d", &diskNum)
		if diskNum > -1 && diskNum < len(disks) {
			break
		}
	}
	fmt.Printf("The backup disk you choose is:%s\r\n", disks[diskNum].Path)

	var err error
	if selfDev == disks[diskNum].Path {
		err = back.Backup("", time.Now().Format("2006-01-02150405")+".img", true)
	} else {
		err = back.Backup(disks[diskNum].Path, time.Now().Format("2006-01-02150405")+".img", false)
	}
	if err != nil {
		fmt.Printf("back err:%v\r\n", err)
	} else {
		fmt.Printf("back ok\r\n")
	}
}
