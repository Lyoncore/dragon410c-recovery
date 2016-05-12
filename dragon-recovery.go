package main

import (
	"log"
	"os"
)

import rplib "github.com/Lyoncore/ubuntu-recovery-rplib"
import rpoem "github.com/Lyoncore/ubuntu-recovery-rpoem"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Recovery snappy system for dragonboard410c")

	rpoem.InitProject("dragonboard410c")

	factoryInstallHook := "factory-install"
	factoryRestoreHook := "factory-restore"
	dataWipeHook := "data-wipe"

	log.Printf("Run hooks in gadget snap: %s, %s, %s", factoryInstallHook, factoryRestoreHook, dataWipeHook)
	hookPath := "/recovery/magic/hooks/"

	_, err := os.Stat(hookPath + factoryInstallHook)
	if err != nil {
		log.Println("Error: can not find " + hookPath + factoryInstallHook)
		os.Exit(1)
	}
	rplib.Shellexec(hookPath + factoryInstallHook)

	rplib.Sync()
	log.Println("Recovery done, reboot")
	rplib.Reboot()
}
