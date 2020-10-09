// +build !windows

package back

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"pi-image/comm"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/juju/utils/v2/fs"
	"github.com/rekby/gpt"
	"github.com/rekby/mbr"
)


func GetDisk() []comm.DiskInfo {
	rd, err := ioutil.ReadDir("/sys/block")
	var disks = make([]comm.DiskInfo, 0)
	var devName = ""
	if err == nil {
		for _, fi := range rd {
			if strings.HasPrefix(fi.Name(), "s") || strings.HasPrefix(fi.Name(), "m") || strings.HasPrefix(fi.Name(), "h") {
				modelBuf, err := ioutil.ReadFile(fmt.Sprintf("/sys/block/%s/device/model", fi.Name()))
				if err == nil {
					devName = strings.TrimSpace(string(modelBuf))
				}
				_disk := comm.DiskInfo{
					Path: fmt.Sprintf("/dev/%s", fi.Name()),
					Name: devName,
				}
				disks = append(disks, _disk)
			}
		}
	}
	return disks
}

func DiskID(devPath string) string {
	b, err := ioutil.ReadFile("/sys/block/" + devPath[strings.LastIndex(devPath, "/"):] + "/device/model") // just pass the file name
	if err != nil {
		return ""
	}
	return string(b)
}

/*
   Device Boot      Start         End      Blocks   Id  System
/dev/sdb1               1       26108   209712478+  83  Linux

*/

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

type partition struct {
	DeviceBoot string
	Start      int64
	End        int64
	Blocks     int64
	Id         int
	System     string
}

func DiskList(devPath string) []partition {
	var partitions = make([]partition, 0)
	var devName = devPath[strings.LastIndex(devPath, "/")+1:]
	var i = 1
	var tmp = ""
	//read MBR
	partTypes := make(map[uint32]mbr.PartitionType)
	f, err := os.Open(devPath)
	defer f.Close()
	if err == nil {
		Mbr, err := mbr.Read(f)
		if err == nil {
			if Mbr.IsGPT() {
				xx1, err := gpt.ReadTable(f, 512)
				fmt.Printf("xx1:%v err:%v\r\n", xx1, err)
			} else {
				for _, part := range Mbr.GetAllPartitions() {
					if part.GetLBAStart() > 0 {
						partTypes[part.GetLBAStart()] = part.GetType()
					}
				}
			}
		} else {
			fmt.Printf("read MBR err:%v\r\n", err)
		}
	} else {
		fmt.Printf("read MBR err2:%v\r\n", err)
	}
	typeSystems := map[int]string{
		0x0c: "W95 FAT32 (LBA)",
		0x07: "NTFS",
		0x82: "linux swap",
		0x83: "linux",
	}

	var Id = 0
	var System = "--"
	var partName = ""
	for {
		partName = ""
		if Exist(fmt.Sprintf("/sys/block/%s/%s%d", devName, devName, i)) {
			partName = fmt.Sprintf("%s%d", devName, i)
		}
		if Exist(fmt.Sprintf("/sys/block/%s/%sp%d", devName, devName, i)) {
			partName = fmt.Sprintf("%sp%d", devName, i)
		}
		if len(partName) == 0 {
			break
		}
		tmp = fmt.Sprintf("/sys/block/%s/%s", devName, partName)
		if !Exist(tmp) {
			break
		}
		//no part
		if !Exist(fmt.Sprintf("%s/partition", tmp)) {
			i++
			continue
		}
		//read start
		var start int64 = 0
		b, err := ioutil.ReadFile(fmt.Sprintf("%s/start", tmp)) // just pass the file name
		if err == nil {
			start, err = strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
			if err != nil {
				fmt.Printf("err:%v\r\n", err)
				start = 0
			}
		}

		//read start
		var end int64 = 0
		b, err = ioutil.ReadFile(fmt.Sprintf("%s/size", tmp)) // just pass the file name
		var blocks int64 = 0
		if err == nil {
			size, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
			if err != nil {
				start = 0
			}
			blocks = size / 2
			end = size + start - 1
		}

		if _, ok := partTypes[uint32(start)]; ok {
			Id = int(partTypes[uint32(start)])
			if _, ok1 := typeSystems[Id]; ok1 {
				System = typeSystems[Id]
			}
		}

		_partition := partition{
			DeviceBoot: fmt.Sprintf("/dev/%s", partName),
			Start:      start,
			End:        end,
			Blocks:     blocks,
			Id:         Id,
			System:     System,
		}
		partitions = append(partitions, _partition)
		i++
	}
	return partitions
}

/*
* 本机备份需要
selfBack true
*/

func Backup(devName string, imgFile string, selfBack bool) error {
	usbmount := "/mnt"
	var err error = nil
	var bootDev = ""
	var rootDev = ""
	//
	if selfBack {
		bootDev = GetSelfBootDev()
		devName = GetSelfDev(bootDev)
	}
	if len(devName) < 1 {
		fmt.Println("devName null, exit now")
		os.Exit(-1)
	}
	var BootMountPath = ""
	var RootMountPath = ""

	var bootStart int64 = 0
	var bootEnd int64 = 0
	var rootStart int64 = 0
	partitions := DiskList(devName)
	fmt.Printf("src dev:%s\r\n", devName)
	//check mounts
	if selfBack {
		BootMountPath = "/boot"
		RootMountPath = "/"
		for _, v := range partitions {
			if v.DeviceBoot == bootDev {
				bootStart = v.Start
				bootEnd = v.End
			}
			if v.Id == 0x83 && rootStart == 0 {
				rootDev = v.DeviceBoot
				rootStart = v.Start
			}
		}
	} else {
		modelBuf, _ := ioutil.ReadFile("/proc/self/mountstats")
		lines := strings.Split(string(modelBuf), "\n")
		mouts := make(map[string]string)
		for _, v := range lines {
			lineInfo := strings.Split(v, " ")
			if len(lineInfo) > 1 {
				mouts[lineInfo[1]] = lineInfo[4]
			}
		}
		var tmpPath = ""
		for _, v := range partitions {
			if v.Blocks > 0 {
				tmpPath = ""
				if v.Id == 0x0c && bootStart == 0 {
					if _, ok := mouts[v.DeviceBoot]; !ok {
						BootMountPath = usbmount + "/sourceBoot"
						tmpPath = BootMountPath
					} else {
						BootMountPath = mouts[v.DeviceBoot]
					}
					bootDev = v.DeviceBoot
					bootStart = v.Start
					bootEnd = v.End
				}
				if v.Id == 0x83 && rootStart == 0 {
					if _, ok := mouts[v.DeviceBoot]; !ok {
						RootMountPath = usbmount + "/sourceRoot"
						tmpPath = RootMountPath
					} else {
						RootMountPath = mouts[v.DeviceBoot]
					}
					rootDev = v.DeviceBoot
					rootStart = v.Start
				}

				//auto mount
				if len(tmpPath) > 0 {
					os.Mkdir(tmpPath, os.ModePerm)
					defer os.RemoveAll(tmpPath)
					err = exec.Command("mount", "-o", "uid=1000", v.DeviceBoot, tmpPath).Run()
					//auto umount
					defer exec.Command("umount", tmpPath).Run()
					if err != nil {
						return err
					}
				}
			}
		}
	}

	fmt.Println("===================== part 1, create a new blank img ===============================")

	if len(BootMountPath) == 0 {
		fmt.Println("Cannot find boot mount directory, exit now")
		os.Exit(-1)
	}
	if len(RootMountPath) == 0 {
		fmt.Println("Cannot find rootfs mount directory, exit now")
		os.Exit(-1)
	}

	fmt.Printf("BootDev:%s RootDev:%s\r\n", bootDev, rootDev)

	fmt.Printf("BootMountPath:%s RootMountPath:%s\r\n", BootMountPath, RootMountPath)

	fmt.Printf("partInfo BootStart:%d BootEnd:%d RootStart:%d\r\n", bootStart, bootEnd, rootStart)

	//获取磁盘大小
	bootInfo := DiskUsage(BootMountPath)
	rootInfo := DiskUsage(RootMountPath)
	zeroSt := time.Now()
	zeroSz := int(float32(bootInfo.All/1024+rootInfo.Used/1024) * 1.3 / 4) //bs=4K  /4
	//err = exec.Command("dd", "if=/dev/zero", "of="+imgFile, "bs=4K", "count="+strconv.Itoa(zeroSz)).Run()
	err = cZeroImgV1(imgFile, uint64(zeroSz), 1024*4)
	fmt.Printf("zero file time:%s\r\n", time.Since(zeroSt))
	if err != nil {
		return err
	}
	err = exec.Command("parted", imgFile, "--script", "--", "mklabel", "msdos").Run()
	if err != nil {
		return err
	}
	err = exec.Command("parted", imgFile, "--script", "--", "mkpart", "primary", "fat32", strconv.Itoa(int(bootStart))+"s", strconv.Itoa(int(bootEnd))+"s").Run()
	if err != nil {
		return err
	}
	err = exec.Command("parted", imgFile, "--script", "--", "mkpart", "primary", "ext4", strconv.Itoa(int(rootStart))+"s", "-1").Run()
	if err != nil {
		return err
	}

	devTmp, err := exec.Command("losetup", "-f", "--show", imgFile).Output()
	if err != nil {
		return err
	}
	devLoop := strings.TrimSpace(string(devTmp))
	//auto remove
	defer exec.Command("losetup", "-d", devLoop).Run()

	err = exec.Command("kpartx", "-va", devLoop).Run()
	if err != nil {
		return err
	}
	defer exec.Command("kpartx", "-d", devLoop).Run()

	///dev/loop1
	device := "/dev/mapper/" + string(devLoop[strings.LastIndex(string(devLoop), "/"):])

	//sleep 5s
	time.Sleep(5 * time.Second)

	err = exec.Command("mkfs.vfat", device+"p1", "-n", "boot").Run()
	if err != nil {
		return err
	}
	err = exec.Command("mkfs.ext4", device+"p2").Run()
	if err != nil {
		return err
	}

	fmt.Println("===================== part 2, fill the data to img =========================")

	var mountb = usbmount + "/backup_boot/"
	var mountr = usbmount + "/backup_root/"

	os.Mkdir(mountb, 0777)
	os.Mkdir(mountr, 0777)

	defer os.RemoveAll(mountb)
	defer os.RemoveAll(mountr)

	//back boot
	err = exec.Command("mount", "-t", "vfat", device+"p1", mountb).Run()
	if err != nil {
		return err
	}
	defer exec.Command("umount", mountb).Run()
	time.Sleep(1 * time.Second)
	err = exec.Command("rsync", "-rt", BootMountPath+"/", mountb).Run()
	if err != nil {
		return err
	}
	err = exec.Command("sync").Run()
	if err != nil {
		return err
	}
	fmt.Println("...Boot partition done")

	//back root
	err = exec.Command("mount", "-t", "ext4", device+"p2", mountr).Run()
	if err != nil {
		return err
	}
	defer exec.Command("umount", mountr).Run()
	time.Sleep(1 * time.Second)
	var excludeDirs []string = []string{".gvfs", "/dev/*", "/media/*", "/mnt/*", "/proc/*", "/run/*", "/sys/*", "/tmp/*", "lost+found/*", ".restoresymtable", usbmount}
	copySt := time.Now()
	//
	if !strings.HasSuffix(RootMountPath, "/") {
		RootMountPath = RootMountPath + "/"
	}
	err = rsyncCopy(RootMountPath, mountr, excludeDirs)
	fmt.Printf("copy time:%s\r\n", time.Since(copySt))
	if err != nil {
		return err
	}

	for _, v := range excludeDirs {
		if Exist(mountr + v) {
			os.RemoveAll(mountr + v)
		}
	}

	err = exec.Command("sync").Run()
	if err != nil {
		return err
	}
	fmt.Println("...Root partition done")

	//replace uuid
	srcBootUuid := GetUUIDByBlkid(bootDev)
	srcRootUuid := GetUUIDByBlkid(rootDev)

	dstBootUuid := GetUUIDByBlkid(device + "p1")
	dstRootUuid := GetUUIDByBlkid(device + "p2")
	fmt.Printf("srcBootUuid:%s srcRootUuid:%s dstBootUuid:%s dstRootUuid:%s\r\n", srcBootUuid, srcRootUuid, dstBootUuid, dstRootUuid)

	Sed(mountb+"cmdline.txt", srcRootUuid, dstRootUuid)
	Sed(mountr+"etc/fstab", srcBootUuid, dstBootUuid)
	Sed(mountr+"etc/fstab", srcRootUuid, dstRootUuid)
	Sed(mountr+"etc/.fstab", srcBootUuid, dstBootUuid)
	Sed(mountr+"etc/.fstab", srcRootUuid, dstRootUuid)

	return nil
}

func cZeroImg(fPath string, fSize uint64) error {
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(int64(fSize)); err != nil {
		return nil
	}
	return nil
}
func cZeroImgV1(fPath string, fSize uint64, bs int) error {
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var buf = make([]byte, bs, bs)
	for i := 0; i < int(fSize); i++ {
		_, err3 := f.Write(buf)
		if err3 != nil {
			return err3
		}
	}
	f.Sync()
	return nil
}

/*rsyncCopy*/
func rsyncCopy(srcDir string, dstDir string, excludeDirs []string) error {
	var excludeDirParams = []string{"--force", "-rltWDEgop", "--delete", "--stats", "--progress"}
	for _, v := range excludeDirs {
		excludeDirParams = append(excludeDirParams, "--exclude", v)
	}
	//add path
	excludeDirParams = append(excludeDirParams, srcDir, dstDir)
	rsyncCmd := exec.Command("rsync", excludeDirParams...)
	rsyncCmd.Stderr = os.Stderr
	err := rsyncCmd.Run()
	if err != nil {
		fmt.Printf("rsync2 err:%v\r\n", err)
		return err
	}
	return err
}

/*dumpCopy*/
func dumpCopy(srcDir string, dstDir string, excludeDirs []string) error {
	c1 := exec.Command("dump", "-0uaf", "-", srcDir)
	c2 := exec.Command("restore", "-rf", "-")
	c2.Dir = dstDir
	c2.Stdin, _ = c1.StdoutPipe()
	c2.Stdout = os.Stdout
	err := c2.Start()
	if err != nil {
		return err
	}
	err = c1.Run()
	if err != nil {
		return err
	}
	err = c2.Wait()
	return err
}

/*tarCopy*/
func tarCopy(srcDir string, dstDir string, excludeDirs []string) error {
	var excludeDirParams = []string{"cp", "c", "-C", srcDir}
	for _, v := range excludeDirs {
		excludeDirParams = append(excludeDirParams, "--exclude="+v)
	}
	c1 := exec.Command("tar", excludeDirParams...)
	c2 := exec.Command("tar", "xvp", "-C", dstDir)

	c2.Stdin, _ = c1.StdoutPipe()
	c2.Stdout = os.Stdout
	err := c2.Start()
	if err != nil {
		return err
	}
	err = c1.Run()
	if err != nil {
		return err
	}
	err = c2.Wait()
	return err
}

func goCopy(srcDir string, dstDir string) error {
	return fs.Copy(srcDir, dstDir)
}

func CpCopy() {
	fmt.Printf("dd")
}

func Sed(file string, old string, newStr string) error {
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if len(old) > 0 {
		xx := strings.Replace(string(input), old, newStr, -1)
		return ioutil.WriteFile(file, []byte(xx), 0644)
	}
	return nil
}

func GetSelfDev(bootPath string) string {
	disks := GetDisk()
	for _, v := range disks {
		if strings.Contains(bootPath, v.Path) {
			return v.Path
		}
	}
	return ""
}

func GetSelfBootDev() string {
	modelBuf, _ := ioutil.ReadFile("/proc/self/mountstats")
	lines := strings.Split(string(modelBuf), "\n")
	for _, line := range lines {
		lineInfo := strings.Split(line, " ")
		if len(lineInfo) > 1 {
			if lineInfo[4] == "/boot" {
				return lineInfo[1]
			}
		}
	}
	return ""
}

func GetUUid(devPath string) string {
	rd, err := ioutil.ReadDir("/dev/disk/by-uuid")
	if err == nil {
		for _, fi := range rd {
			path, err := os.Readlink("/dev/disk/by-uuid/" + fi.Name())
			if err == nil {
				if path[strings.LastIndex(path, "/"):] == devPath[strings.LastIndex(devPath, "/"):] {
					return fi.Name()
				}
			}
		}
	}
	return ""
}
func GetUUIDByBlkid(devPath string) string {
	out, err := exec.Command("blkid", "-o", "export", devPath).Output()
	if err != nil {
		return ""
	}
	if err == nil {
		outInfo := strings.Split(string(out), "\n")
		for _, line := range outInfo {
			if strings.HasPrefix(line, "PARTUUID=") {
				return line[strings.Index(line, "PARTUUID=")+9:]
			}
		}
	}
	return ""
}

type DiskStatus struct {
	All  uint64
	Used uint64
	Free uint64
}

// disk usage of path/disk
func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks * uint64(fs.Bsize)
	disk.Free = fs.Bfree * uint64(fs.Bsize)
	disk.Used = disk.All - disk.Free
	return
}

func CheckPm(){
	//check root
	if os.Geteuid() != 0 {
		fmt.Printf("Please run with sudo.\r\n")
		os.Exit(-1)
	}
}