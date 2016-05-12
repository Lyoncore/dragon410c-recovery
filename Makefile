all:
	go build dragon-recovery-image.go
	GOARCH=arm64 go build dragon-recovery.go
	mkenvimage -r -s 131072 -o uboot.env uboot.env.in
	cat canonical-snapdragon-linux_5.snapaa canonical-snapdragon-linux_5.snapab > canonical-snapdragon-linux_5.snap
	cat dragon-all-snap.img.xzaa dragon-all-snap.img.xzab > dragon-all-snap.img.xz
	sudo ./dragon-recovery-image

clean:
	rm -f factory_data recovery.squashfs
