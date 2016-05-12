package main

import (
	"io/ioutil"
	"log"
	"os"
	"fmt"
	"bufio"
	"strconv"
	"strings"
)

import flags "github.com/jessevdk/go-flags"
import rplib "github.com/Lyoncore/ubuntu-recovery-rplib"
import rpoem "github.com/Lyoncore/ubuntu-recovery-rpoem"

// Options store all arguments from command line
type Options struct {
	Kernelsnap   string `short:"k" long:"kernel" default:"canonical-snapdragon-linux_5.snap" description:"Kernel snap"`
	OSsnap string `short:"o" long:"os" default:"ubuntu-core_111.snap" description:"OS snap"`
	Gadgetsnap string `short:"g" long:"gadget" default:"canonical-dragon_5.snap" description:"Gadget snap"`
	Initrd string `short:"i" long:"initrd" default:"initrd.img" description:"Recovery initrd image"`
	Vmlinux string `short:"v" long:"vmlinux" default:"vmlinuz" description:"vmlinux.bin"`
}

var globalOpt Options
var parser = flags.NewParser(&globalOpt, flags.HelpFlag)

func createRecoveryImage(device string, RECOVERY_LABEL string, tempdir string) {
        var err error
        OLD_PARTITION := fmt.Sprintf("%s/old-partition.txt", tempdir)
        boot_begin := -1
        boot_size := -1
        writable_nr := -1
        writable_begin := -1
        writable_size := -1

        rplib.Shellcmd(fmt.Sprintf("parted -ms %s unit B print | sed -n '1,2!p' > %s", device, OLD_PARTITION))

        // Read information of partitions
        var f *(os.File)
        f, err = os.Open(OLD_PARTITION)
        rplib.Checkerr(err)
        scanner := bufio.NewScanner(f)
        for scanner.Scan() {
                line := scanner.Text()
                log.Println("line: ", line)
                fields := strings.Split(line, ":")
                log.Println("fields: ", fields)
                nr, err := strconv.Atoi(fields[0])
                rplib.Checkerr(err)
                begin, err := strconv.Atoi(strings.TrimRight(fields[1], "B"))
                end, err := strconv.Atoi(strings.TrimRight(fields[2], "B"))
                rplib.Checkerr(err)
                size,err := strconv.Atoi(strings.TrimRight(fields[3], "B"))
                fstype := fields[4]
                label := fields[5]
                log.Println("nr: ", nr)
                log.Println("begin: ", begin)
                log.Println("end: ", end)
                log.Println("size: ", size)
                log.Println("fstype: ", fstype)
                log.Println("label: ", label)

                if label == "system-boot" {
			boot_begin = begin
                        boot_size = size
                        log.Println("boot_begin:", boot_begin)
                        log.Println("boot_size:", boot_size)
                }
                if label == "writable" {
                	writable_nr = nr
			writable_begin = begin
			writable_size = size
                        log.Println("writable_begin:", writable_begin)
                        log.Println("writable_size:", writable_size)
                }
        }
        err = scanner.Err()
        rplib.Checkerr(err)
        
        //update system-boot
        bootimgFilename := fmt.Sprintf("%s/system-boot.img", tempdir)
        
        rplib.Shellexec("dd", fmt.Sprintf("if=%s", device), fmt.Sprintf("of=%s", bootimgFilename), "bs=1M", 
        	fmt.Sprintf("skip=%d", boot_begin), 
        	fmt.Sprintf("count=%d", boot_size), "iflag=count_bytes,skip_bytes", "oflag=seek_bytes", "conv=notrunc")

	systembootDir := fmt.Sprintf("%s/mount-system-boot-dir", tempdir)
	err = os.Mkdir(systembootDir, 0777)
	if err != nil {
		log.Println("Failed to create directory for sysbootdir")
		log.Fatal(err)
	}
        	
	rplib.Shellexec("mount",bootimgFilename,systembootDir)
	rplib.Shellexec("cp","-f","uboot.env",systembootDir)
	rplib.Shellexec("umount",systembootDir)
        
        //resize writable
        const recoverySizeM = 768
        resizedFilename := fmt.Sprintf("%s/writable.resize", tempdir)
        allimage := fmt.Sprintf("%s/new-all.img", tempdir)
        
        rplib.Shellexec("dd", fmt.Sprintf("if=%s", device), fmt.Sprintf("of=%s", resizedFilename), "bs=1M", 
        	fmt.Sprintf("skip=%d", writable_begin), 
        	fmt.Sprintf("count=%d", writable_size), "iflag=count_bytes,skip_bytes", "oflag=seek_bytes", "conv=notrunc")
	rplib.Shellexec("e2fsck", "-fy", resizedFilename)
	
	resizeSizeM := (writable_size- (recoverySizeM * 1024 * 1024)) / 1024 / 1024
        resizeSizeMString := fmt.Sprintf("%dM", resizeSizeM)

        rplib.Shellexec("resize2fs", resizedFilename, resizeSizeMString)
        rplib.Shellcmd(fmt.Sprintf("cat %s %s > %s","factory_data", resizedFilename, allimage))

        //recreate partition
        rplib.Shellexec("sgdisk", "-d", fmt.Sprintf("%v", writable_nr), device)
	rplib.Shellexec("sgdisk", "-n", "0:0:+768M","-c","0:factory-data", device )
	rplib.Shellexec("sgdisk", "-n", "0:0:0", "-c", "0:writable", device)
	
	//dd the factory-data
	rplib.Shellexec("dd", fmt.Sprintf("if=%s", allimage), fmt.Sprintf("of=%s", device), "bs=1M", 
                fmt.Sprintf("seek=%d", writable_begin), 
                "iflag=count_bytes,skip_bytes", "oflag=seek_bytes", "conv=notrunc")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if _, err := parser.ParseArgs(os.Args); err != nil {
		log.Fatal(err)
	}

	rpoem.InitProject("dragon410c")

	log.Println("Build recovery snap for Dragonboard 410c")
	
	//create temp working directory

	log.Println("Create a temporary directory")
	tmpdir, err := ioutil.TempDir("/precise/tmp", "")
	if err != nil {
		log.Println("Failed to create temporary folder")
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	//create recovery/factory partition.image

	log.Println("Make recovery partition image")
	factory_data := "factory_data"
	rplib.Shellexec("rm", "-rf", factory_data)
	rplib.Shellexec("dd", "if=/dev/zero", "of="+factory_data, "bs=32M", "count=24")
	rplib.Shellexec("mkfs.ext4", "-F", factory_data)
	rplib.Shellexec("e2label", factory_data, rpoem.FilesystemLabel())

	log.Println("Create a directory for the factory partition")
	factoryNewDir := tmpdir + "/factory_new"
	err = os.Mkdir(factoryNewDir, 0777)
	if err != nil {
		log.Println("Failed to create directory for factory partition")
		log.Fatal(err)
	}

	//TIM: create factory partition
	rplib.Shellexec("mount", factory_data, factoryNewDir)

	log.Println("mount directories for the factory partition")
	KernelDir := tmpdir + "/" + globalOpt.Kernelsnap
	err = os.Mkdir(KernelDir, 0777)
	if err != nil {
		log.Println("Failed to create directory for kernelsnap-directory")
		log.Fatal(err)
	}
	rplib.Shellexec("mount", globalOpt.Kernelsnap, KernelDir)

	log.Println("Copy kernel snaps contents to " + factoryNewDir)
	rplib.Shellexec("cp", "-af", KernelDir, factoryNewDir)
	
	//add other stuffs
	//add vmlinuz
	rplib.Shellexec("cp", "-fL", KernelDir + "/" + globalOpt.Vmlinux, factoryNewDir)

	//add initrd
	initrdImg := tmpdir + "/" + globalOpt.Initrd
	initrdLzma := initrdImg + ".lzma"
	rplib.Shellexec("cp", "-fL", KernelDir + "/" + globalOpt.Initrd, initrdLzma)
	//add dtb
	log.Println("Copy dtb to " + factoryNewDir)
	rplib.Shellexec("cp", "-af", KernelDir + "/dtbs", factoryNewDir)

	rplib.Shellexec("umount", KernelDir)

	GadgetDir := tmpdir + "/" + globalOpt.Gadgetsnap
	err = os.Mkdir(GadgetDir, 0777)
	if err != nil {
		log.Println("Failed to create directory for gadgetsnap-directory")
		log.Fatal(err)
	}
	rplib.Shellexec("mount", globalOpt.Gadgetsnap, GadgetDir)
	log.Println("Copy gadget snaps contents to " + factoryNewDir)
	rplib.Shellexec("cp", "-af", GadgetDir, factoryNewDir)

	rplib.Shellexec("umount", GadgetDir)
	
	//dragonboard410c 
	log.Println("Copy system-boot stuff to" + factoryNewDir)
	rplib.Shellexec("cp", "-af", GadgetDir, factoryNewDir)

        SnapsDir := factoryNewDir + "/snaps"
        err = os.Mkdir(SnapsDir, 0777)
        if err != nil {
                log.Println("Failed to create directory for snaps dir")
                log.Fatal(err)
        }

	log.Println("Copy snaps to " + SnapsDir)
	rplib.Shellexec("cp", "-f", globalOpt.OSsnap, SnapsDir)
	rplib.Shellexec("cp", "-f", globalOpt.Kernelsnap, SnapsDir)
	rplib.Shellexec("cp", "-f", globalOpt.Gadgetsnap, SnapsDir)

	log.Println("Unlzma " + initrdLzma)
	rplib.Shellexec("unlzma", initrdLzma)

	log.Println("Create a directory for the new initrd")
	initrdNewDir := tmpdir + "/initrd_new"
	err = os.Mkdir(initrdNewDir, 0777)
	if err != nil {
		log.Println("Failed to create directory for new initrd")
		log.Fatal(err)
	}

	log.Println("Extract CPIO file")
	initrdImg = tmpdir + "/" + globalOpt.Initrd
	rplib.Shellcmd("cd " + initrdNewDir + " && cpio -i --no-absolute-filenames < " + initrdImg)

	log.Println("Copy 00_recovery to " + initrdNewDir + "/scripts/local-premount/")
	rplib.Shellexec("cp", "-f", "00_recovery", initrdNewDir+"/scripts/local-premount/")

	log.Println("Modify " + initrdNewDir + "/scripts/local-premount/ORDER")
	rplib.Shellcmd("echo '/scripts/local-premount/00_recovery \"$@\"' >> " + initrdNewDir + "/scripts/local-premount/ORDER")
	rplib.Shellcmd("echo '[ -e /conf/param.conf ] && . /conf/param.conf' >> " + initrdNewDir + "/scripts/local-premount/ORDER")
	
	log.Println("Create a recovery directory for the new initrd")
        err = os.Mkdir(initrdNewDir + "/recovery", 0777)
        if err != nil {
                log.Println("Failed to create recovery directory for new initrd")
                log.Fatal(err)
        }

	log.Println("Create new initrd")
	rplib.Shellcmd("cd " + initrdNewDir + " && find . | cpio --quiet -R 0:0 -o -H newc | lzma > " + factoryNewDir + "/" + globalOpt.Initrd)

	rplib.Shellexec("umount", factoryNewDir)
	
	//insert this partition image to base image
	rplib.Shellcmd("xz -dcfk dragon-all-snap.img.xz > dragon-recovery.img")
	createRecoveryImage("dragon-recovery.img","factory-data",tmpdir)
	
}
 