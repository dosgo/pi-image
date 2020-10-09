package comm

import (
	"fmt"
	"os"
	"strings"
)

type DiskInfo struct {
	Path string
	Name string
}


func CheckCmd() {
	cmds := map[string]string{
		//"dd":        "",
		"parted":    "sudo apt-get install parted",
		"losetup":   "",
		"dump":      "sudo apt-get install dump",
		"restore":   "sudo apt-get install dump",
		"tar":       "sudo apt-get install tar",
		"kpartx":    "sudo apt-get install kpartx",
		"mkfs.vfat": "",
		"mkfs.ext4": "",
		"mount":     "",
		"rsync":     "sudo apt-get install rsync",
		"umount":    "",
		"blkid":     "sudo apt-get install blkid",
	}
	for k, v := range cmds {
		if !which(k) {
			fmt.Printf("%s: command not found\r\n", k)
			fmt.Printf("%s\r\n", v)
			os.Exit(-1)
		}
	}
}

/*
*https://github.com/clibs/which/blob/master/src/which.c
 */
func which(name string) bool {
	pathStr := os.Getenv("PATH")
	paths := strings.Split(pathStr, ":")
	for _, v := range paths {
		file_info, err := os.Stat(fmt.Sprintf("%s/%s", v, name))
		if err != nil {
			continue
		}
		flag := file_info.Mode().Perm() & os.FileMode(73)
		if uint32(flag) == uint32(73) {
			return true
		}
	}
	return false
}
