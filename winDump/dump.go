package winDump

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/rekby/mbr"
	"golang.org/x/sys/windows"
	"io"
	"log"
	"math"
	"os"
	"pi-image/win"
	"strconv"
	"unsafe"
)

var MBRLEN=512;
var BUFFER_SIZE=9216



func writeZero( fp *os.File,start uint64, _len uint64) {
	var buf = make([]byte, 512, 512)
	if(_len>0) {
		fp.Seek(int64(start)*512, io.SeekCurrent)
		for i := 0; i < int(_len); i++ {
			fp.Write(buf);
		}
	}
}

func writeImg( hDevice windows.Handle,fp *os.File , startPos uint64, endPos int) {
	var  outBuf string;
	var  i int=0
	var endlen uint32=0
	var buffer = make([]byte, BUFFER_SIZE)
	var  forLenF float64=float64(endPos*512/BUFFER_SIZE);
	var forlen int=int(math.Ceil(forLenF));
	var offsetBytes = make([]byte, 8)
	fp.Seek(int64(startPos)*512,io.SeekCurrent)
	binary.LittleEndian.PutUint64(offsetBytes, uint64(startPos*512+uint64(i)*uint64(BUFFER_SIZE)))
	lowoffset := *(*int32)(unsafe.Pointer(&offsetBytes[0]))
	highoffsetptr := (*int32)(unsafe.Pointer(&offsetBytes[4]))
	windows.SetFilePointer(hDevice,lowoffset,highoffsetptr, windows.FILE_CURRENT);

	for i=0;i<forlen;i++{

		if((i+1)==forlen) {
			endlen=uint32(endPos*512-BUFFER_SIZE*i);
		}else {
			endlen=uint32(BUFFER_SIZE);
		}
		err:=windows.ReadFile(hDevice, buffer, &endlen,nil);
		if(err==nil) {
			_,err=fp.Write(buffer);
			if(err!=nil){
				fmt.Printf("write img error\r\n");
			}
		}
		outBuf=fmt.Sprintf("%.2f", float32(i*100.0/forlen));
		// fseek(stdout,-7,SEEK_CUR);
		fmt.Printf(outBuf);
		fmt.Printf("%s","%");
		//这是重点
		for j:=0;j<=len(outBuf);j++{
			fmt.Printf("\r");
		}

	}
	fmt.Printf("100%s ok \r\n","%");

}




func RestoreImg( volume string,savepath string ) int{
	//打开卷
	var volumename =fmt.Sprintf("\\\\.\\%c:",volume);
	hVolume,err:= windows.CreateFile(windows.StringToUTF16Ptr(volumename), windows.GENERIC_WRITE, windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0);
	if err!=nil {
		log.Printf("打开卷失败\r\n");
		return -1;
	}
	//锁定卷
	var  bytesreturned uint32;
	bResult:= windows.DeviceIoControl(hVolume, win.FSCTL_LOCK_VOLUME, nil, 0, nil, 0, &bytesreturned, nil);
	if (bResult!=nil) {
		windows.CloseHandle(hVolume);
		log.Printf("锁定卷失败\r\n");
		return -1;
	}
	//卸载卷
	var  junk uint32;
	bResult= windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
	if (bResult!=nil) {
		//移除锁定
		windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
		windows.CloseHandle(hVolume);
		log.Printf("卸载卷失败\r\n");
		return -1;
	}
	//获取卷的磁盘号
	var readed uint32;


	size := uint32(16 * 1024)
	vde := make(win.VolumeDiskExtents, size)

	err=windows.DeviceIoControl(hVolume,win.IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS,nil,0,&vde[0],size,&readed,nil);


	var dwDiskNumber  = strconv.FormatUint(uint64(vde.Extent(0).DiskNumber), 10)


	var  devicename =fmt.Sprintf("\\\\.\\PhysicalDrive%d",dwDiskNumber);

	 hDevice,err:=windows.CreateFile(windows.StringToUTF16Ptr(devicename),
		windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE,
	nil,
		windows.OPEN_EXISTING,
	0,
	0);

	if  err !=nil   {
		log.Printf("CreateFile Error %d \r\n", windows.GetLastError());
		return -1;
	}




	//打开镜像
	fp,err:=os.OpenFile(savepath,os.O_RDONLY,0777)
	defer fp.Close();
	if(err!=nil) {
		log.Printf("open img error\r\n");
		return -1;
	}
	var  buffer =make([]byte,BUFFER_SIZE);
	var  len int=0;
	var byteswritten uint32;
	var ulen uint32;



	for {
		len,err= fp.Read(buffer)
		if len>0{
			ulen=uint32(len);
			err = windows.WriteFile(hDevice, buffer, &ulen,nil);
			if (err!=nil&&err!= windows.ERROR_IO_PENDING){
				log.Printf("Restore error...:%d\r\n", windows.GetLastError());
				break;
			}
			log.Printf("byteswritten:%d\r\n", byteswritten);
		}else{
			break;
		}
	}



	fp.Close();
	log.Printf("Restore ok...\r\n");
	//解除锁定
	windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
	windows.CloseHandle(hVolume);
	windows.CloseHandle(hDevice);
	return 0;
}


//
func DumpImg(volume string,savepath string) int {
	//打开卷
	var  volumeName =fmt.Sprintf("\\\\.\\%s:",volume);
	hVolume,err := windows.CreateFile(windows.StringToUTF16Ptr(volumeName), windows.GENERIC_READ, windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0);
	if (err!=nil) {
		log.Printf("Failed to open volume. volumeName:%s\r\n",volumeName);
		return -1;
	}
	defer  windows.CloseHandle(hVolume);
	//锁定卷
	var  bytesreturned uint32;
	//FSCTL_DISMOUNT_VOLUME
	 bResult:= windows.DeviceIoControl(hVolume, win.FSCTL_LOCK_VOLUME, nil, 0, nil, 0, &bytesreturned, nil);
	if (bResult!=nil) {
		windows.CloseHandle(hVolume);
		log.Printf("Failed to lock volume.\r\n");
		return -1;
	}
	//卸载卷
	var  junk uint32;
	bResult = windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
	if (bResult!=nil) {
		//移除锁定
		windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
		windows.CloseHandle(hVolume);
		log.Printf("Failed to unlock volume.\r\n");
		return -1;
	}
	//获取卷的磁盘号
	var  readed uint32;

//	VOLUME_DISK_EXTENTS vde;
	size := uint32(16 * 1024)
	vde := make(win.VolumeDiskExtents, size)


	err=windows.DeviceIoControl(hVolume,win.IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS,nil,0,&vde[0],size,&readed,nil);
	if err != nil {
		return -1
	}
	if vde.Len() != 1 {
		fmt.Errorf("could not identify physical drive for %s", volume)
		return -1;
	}
	var dwDiskNumber = strconv.FormatUint(uint64(vde.Extent(0).DiskNumber), 10)


	var  devicename =fmt.Sprintf("\\\\.\\PhysicalDrive%s",dwDiskNumber);

	 hDevice,err:= windows.CreateFile(windows.StringToUTF16Ptr(devicename),
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE,
	nil,
		windows.OPEN_EXISTING,
	0,
	0);

	//if ( err == windows.INVALID_HANDLE_VALUE )
	if ( err !=nil) {
		log.Printf("CreateFile Error %d \r\n",windows.GetLastError());
		return -1;
	}
	defer  windows.CloseHandle(hDevice);
	//读取MBR
	var  MbrBuf =make([]byte,512);
	var len uint32=uint32(MBRLEN);
	err=windows.ReadFile(hDevice, MbrBuf, &len,nil);

	if(err!=nil){
		log.Printf("read Mbr Error %d \r\n",windows.GetLastError());
		return -1;
	}

	//打开镜像
	//FILE* fp = fopen(savepath,"wb");
	fp,err:=os.OpenFile(savepath,os.O_WRONLY | os.O_CREATE,0777)
	if(err!=nil) {
		fmt.Printf("err:%v savepath:%s\r\n",err,savepath)
		log.Printf("dump  error 5\r\n");
		return -1;
	}
	defer  fp.Close();
	//写MBR
	fp.Write(MbrBuf)




	Mbr, err := mbr.Read(bytes.NewBuffer(MbrBuf))
	if(Mbr.IsGPT()){
		log.Printf("GPT disks are not supported.\r\n");
		os.Exit(-1);
	}

	fmt.Printf("dump  start\r\n");
    partitions:=Mbr.GetAllPartitions();
    var i=1;
    var last=0;
	for _, partition := range  partitions{
		//有起始偏移量，表示分区存在
		if(partition.GetType()!=0){
			//写空闲的
			writeZero(fp,uint64(last),uint64(partition.GetLBAStart()-uint32(last)));
			fmt.Printf("dump partition %d  \r\n",i);
			writeImg(hDevice,fp,uint64(partition.GetLBAStart()),int(partition.GetLBALen()));
		}
		fmt.Printf("partition:%d\r\n",i)
		i++;
	}
	fmt.Printf("dump ok...\r\n");
	//解除锁定
	windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
	return 0;
}






func test( argc int ,argv []string) int{

/*
   if(IsProcessRunAsAdmin()==FALSE){
       MessageBox(NULL,"警告","请用管理员身份运行本程序",MB_OK);
       return -1;
   }*/
//     RestoreImg('H',"e:\\dumptest.img");


	if(argc>1&&argv[1]=="dump"){
		DumpImg(argv[2],argv[3]);
	}else if(argc>1&&argv[1]=="restore"){
		RestoreImg(argv[2],argv[3]);
	//sddump3.exe restore H e:\dumptest.img
	} else{
			log.Printf( "use " );
			log.Printf("%s",argv[0]);
			log.Printf( "  dump  I  e:\\ddd.img \r\n" );
	}
	return 0;
}
