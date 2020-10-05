package oldDump

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/windows"
	"io/ioutil"
	"log"
	"math"
	"os"
	"pi-image/win"
	"strconv"
	"unsafe"
)

const  BOOTRECORDSIZE =440
const BUFFER_SIZE  =9216
const  DPTSIZE= 64
const DPTNUMBER=4
const MBRLEN=512;

type  BOOTRECORD struct
{
	BootRecord  [BOOTRECORDSIZE]byte
};



type  DPT struct
{
	Dpt  [DPTSIZE]byte
}



type  PDP struct
{
 BootSign uint8;         // 引导标志
 StartHsc [3]byte;
 PartitionType uint8;    // 分区类型
 EndHsc [3]byte ;
 SectorsPreceding uint64;     // 本分区之前使用的扇区数
 SectorsInPartition uint64;   // 分区的总扇区数
}

type  MBR struct
{
	BootRecord BOOTRECORD ;                  // 引导程序
	ulSigned [4]byte;              // Windows磁盘签名
	sReserve [2]byte;              // 保留位
	Dpt DPT ;                         // 分区表
	EndSign [2]byte;                // 结束标志
}

// 显示MBR数据
func ShowMbr(hDevice windows.Handle, pMbr MBR ) {
	var mbr = make([]byte, 512)
	var done uint32 = uint32(MBRLEN)
	windows.ReadFile(hDevice, mbr, &done, nil);
	for  i:= 0; i < 512; i ++{
		log.Printf("%02X ", mbr[i]);
		if  ( i + 1 ) % 16 == 0 {
			log.Printf("\r\n");
		}
	}
}

// 解析MBR
func ParseMbr( Mbr MBR) {
	log.Printf("引导记录: \r\n");

	for  i:= 0; i < BOOTRECORDSIZE; i ++{
		log.Printf("%02X ", Mbr.BootRecord.BootRecord[i]);
		if  ( i + 1 ) % 16 == 0 {
			log.Printf("\r\n");
		}
	}

	log.Printf("\r\n");

	log.Printf("磁盘签名: \r\n");
	for i:= 0; i < 4; i ++{
		log.Printf("%02X ", Mbr.ulSigned[i]);
	}

	log.Printf("\r\n");

	log.Printf("解析分区表: \r\n");
	for i:= 0; i < DPTSIZE; i ++{
		log.Printf("%02X ", Mbr.Dpt.Dpt[i]);
		if ( i + 1 ) % 16 == 0 {
			log.Printf("\r\n");
		}
	}

	log.Printf("\r\n");

	//Mbr.Dpt.Dpt


	var pDp *PDP = &PDP{}


	//var  pDp PDP= Mbr.Dpt.Dpt;
	for  i:= 0; i < DPTNUMBER; i ++{

		binary.Read(bytes.NewBuffer(Mbr.Dpt.Dpt[(i-1)*16:(i-1)*16+16]), binary.LittleEndian, &pDp)

		log.Printf("引导标志: %02X ", pDp.BootSign);
		log.Printf("分区类型: %02X", pDp.PartitionType);
		log.Printf("\r\n");
		log.Printf("本分区之前扇区数: %d ", pDp.SectorsPreceding);
		log.Printf("本分区的总扇区数: %d", pDp.SectorsInPartition);
		log.Printf("\r\n");
		log.Printf("该分区的大小: %f \r\n", pDp.SectorsInPartition / 1024 * 512 / 1024 / 1024 );
		log.Printf("\r\n \r\n");
	}

	log.Printf("结束标志: \r\n");
	for i:= 0; i < 2; i ++{
		log.Printf("%02X ", Mbr.EndSign[i]);
	}

	log.Printf("\r\n");
}



func copyFile(sourceFile string, destinationFile string) int{
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		fmt.Println(err)
		return -1
	}

	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		fmt.Println("Error creating", destinationFile)
		fmt.Println(err)
		return -1
	}

	return 0;
}

func writezero( fp *os.File, len uint64) {
	var zero =make([]byte,1);
	zero[0]=0;
	for i:=0;i<int(len);i++{
		fp.Write(zero);
	}
}
/*
#include "BigNum.h"
//指数转整数
double mypow(float x,int y){
int y1=1;
for(int i=0;i<y;i++)
{
y1=y1*10;
}
return x*y1;
}

#define _FILE_OFFSET_BITS 64
*/
func writeimg( hDevice windows.Handle,fp *os.File , startpos uint64, endpos int) {
	var  outBuf string;
	var  bufsize int=102400;
	var  i int=0
	var endlen uint32=0
	var buffer = make([]byte, bufsize)
	var  forLenF float64=float64(endpos*512/bufsize);
	var forlen int=int(math.Ceil(forLenF));
	stdOut := bufio.NewWriter(os.Stdout)
	//LARGE_INTEGER li;
	for i=0;i<forlen;i++{
		//pos=ftello64(fp);
		offset,err:=fp.Seek(0,1)
		if(err!=nil){
			stdOut.Write([]byte("seek error\r\n"));
			break;
		}

		//li.QuadPart = pos;
		offsetBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(offsetBytes, uint64(offset))
		lowoffset := *(*int32)(unsafe.Pointer(&offsetBytes[0]))
		highoffsetptr := (*int32)(unsafe.Pointer(&offsetBytes[4]))

		windows.SetFilePointer(hDevice,lowoffset,highoffsetptr, windows.FILE_BEGIN);
		if((i+1)==forlen) {
			endlen=uint32(endpos*512-bufsize*i);
		}else {
			endlen=uint32(bufsize);
		}
		err=windows.ReadFile(hDevice, buffer, &endlen,nil);
		if(err==nil) {
			_,err=fp.Write(buffer);
			if(err!=nil){
				stdOut.Write([]byte("write img error\r\n"));
			}
		}
		fmt.Sprintf(outBuf,"%.2lf", i* 100.0/forlen);
		// fseek(stdout,-7,SEEK_CUR);
		stdOut.Write([]byte(outBuf));
		stdOut.Write([]byte("%%"));
		//这是重点
		for j:=0;j<=len(outBuf);j++{
			stdOut.Write([]byte("\b"));
		}
		stdOut.Flush()
	}
	stdOut.Write([]byte("100%% ok \n"));
}




func RestoreImg( volume string,savepath string ) int{
	//打开卷
	var volumename string;
	fmt.Sprintf(volumename,"\\\\.\\%c:",volume);
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


	var  devicename string;
	fmt.Sprintf(devicename,"\\\\.\\PhysicalDrive%d",dwDiskNumber);

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
func DumpImg( volume string,savepath string) int {
	//打开卷
	var  volumename string
	fmt.Sprintf(volumename,"\\\\.\\%c:",volume);
	 hVolume,err := windows.CreateFile(windows.StringToUTF16Ptr(volumename), windows.GENERIC_READ, windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0);
	if (err!=nil) {
		log.Printf("打开卷失败\r\n");
		return -1;
	}
	//锁定卷
	var  bytesreturned uint32;
	//FSCTL_DISMOUNT_VOLUME
	 bResult:= windows.DeviceIoControl(hVolume, win.FSCTL_LOCK_VOLUME, nil, 0, nil, 0, &bytesreturned, nil);
	if (bResult!=nil) {
		windows.CloseHandle(hVolume);
		log.Printf("锁定卷失败\r\n");
		return -1;
	}
	//卸载卷
	var  junk uint32;
	bResult = windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
	if (bResult!=nil) {
		//移除锁定
		windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
		windows.CloseHandle(hVolume);
		log.Printf("卸载卷失败\r\n");
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


	var  devicename string;
	fmt.Sprintf(devicename,"\\\\.\\PhysicalDrive%s",dwDiskNumber);

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
	fp,err:=os.OpenFile(savepath,os.O_WRONLY,0777)
	if(err!=nil) {
		log.Printf("dump  error 5\r\n");
		return -1;
	}

	//写MBR
	fp.Write(MbrBuf)

	//解析mbr,小端
	var Mbr *MBR = &MBR{}
	binary.Read(bytes.NewBuffer(MbrBuf), binary.LittleEndian, &Mbr)

	
	var pDp *PDP = &PDP{}


	//上一扇区
	var  last uint64 =1;
	log.Printf("dump  start\r\n");
	for i:= 0; i < DPTNUMBER; i ++ {
		binary.Read(bytes.NewBuffer(Mbr.Dpt.Dpt[(i-1)*16:(i-1)*16+16]), binary.LittleEndian, &pDp)


		//有起始偏移量，表示分区存在
		if(pDp.SectorsPreceding>0){
			//写空闲的
			log.Printf("dumping partition %d  ",i+1);
			var  pos uint64=(pDp.SectorsPreceding-last)*512;
			writezero(fp,pos);
			writeimg(hDevice,fp,pDp.SectorsPreceding,int(pDp.SectorsInPartition));
			//上一分区最后扇区
			last=pDp.SectorsPreceding+pDp.SectorsInPartition;
		}
	}
	fp.Close()
	log.Printf("dump ok...\r\n");
	//解除锁定
	windows.DeviceIoControl(hVolume, win.FSCTL_UNLOCK_VOLUME, nil, 0, nil, 0, &junk, nil);
	windows.CloseHandle(hVolume);
	windows.CloseHandle(hDevice);
	return 0;
}


func  SizeImg( partitionnum int, newsize int64,filepath  string,newfilepath string) int{

	copyFile(filepath,newfilepath);
	//打开镜像
	fp1,err:=os.OpenFile(newfilepath,os.O_APPEND,0777)
	if(err!=nil){
		return -1;
	}
	defer  fp1.Close();
	var  Partitionsize int64=(newsize*1024*1024)/512;

	//读取MBR
	var  Mbr = MBR{}
	//fread(&Mbr,sizeof(MBR),1,fp1);
	//遍历分区
	var pDp *PDP = &PDP{}
	var  addPartition int=0;
	for i:= 0; i < DPTNUMBER; i ++ {
		binary.Read(bytes.NewBuffer(Mbr.Dpt.Dpt[(i-1)*16:(i-1)*16+16]), binary.LittleEndian, &pDp)
		if(partitionnum==i+1){
			//有起始偏移量，表示分区存在
			if(pDp.SectorsPreceding>0){
				pDp.SectorsPreceding=  pDp.SectorsPreceding+uint64(addPartition);
				//增大
				if(Partitionsize>int64(pDp.SectorsInPartition)) {
					addPartition=int(Partitionsize-int64(pDp.SectorsInPartition));
					var pos uint64=pDp.SectorsInPartition*512;
					fp1.Seek(int64(pos),windows.FILE_BEGIN)
					writezero(fp1,uint64(addPartition*512));
					pDp.SectorsInPartition=uint64(Partitionsize);
				}
			}
		}
	}


	//修改MBR
	//fwrite(&Mbr,sizeof(MBR),1,fp);
//	fp1 = fopen(newfilepath,"rb+");
	//fwrite(&Mbr,sizeof(MBR),1,fp1);
	//fclose(fp1);
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
