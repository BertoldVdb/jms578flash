package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/BertoldVdb/jms578flash/jmshal"
	"github.com/BertoldVdb/jms578flash/scsi"
	"github.com/BertoldVdb/jms578flash/spiflash"
)

func main() {
	bl := flag.Bool("bl", false, "Go to bootloader")
	dev := flag.String("dev", "/dev/sdb", "Device to use")
	on := flag.Bool("on", false, "Turn on led")

	flag.Parse()

	/*	fw, err := ioutil.ReadFile("../../modfw.bin")
		if err != nil {
			log.Fatalln(fw)
		}

		fw2, err := jmshal.PatchFirmware(fw)
		if err != nil {
			log.Fatalln(err, len(fw2))
		}

		ioutil.WriteFile("/home/bertold/vmshared/test2.bin", fw2, 0644)*/

	sdev, err := scsi.New(*dev)
	if err != nil {
		log.Fatalln(err)
	}

	jms, err := jmshal.New(sdev)
	if err != nil {
		log.Fatalln(err)
	}

	jms.XDATAWrite(0x7054, []byte{0xe1})
	if *on {
		jms.XDATAWrite(0x7058, []byte{0x10})
	} else {
		jms.XDATAWrite(0x7058, []byte{0x00})
	}

	return
	//jms.GoROM()

	//if true {
	//bootloader, err := ioutil.ReadFile("../../rom_dump.bin")

	//	log.Println("go patched", jms.GoPatched(bootloader))
	//}
	//tasks, err := jmstasks.New(jms)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//log.Println(tasks)

	//flashfw, err := ioutil.ReadFile("/home/bertold/vmshared/test2.bin")
	//if err != nil {
	//log.Fatalln(err)
	//}
	//log.Println(jms.FlashWriteFirmware(flashfw))
	var bootloader []byte
	if true {
		therom, err := jms.DumpBootrom()
		if err != nil {
			log.Fatalln(err)
		}
		ioutil.WriteFile("/tmp/rom_dump.bin", therom, 0644)
	} else {
		var err error
		bootloader, err = ioutil.ReadFile("/tmp/rom_dump.bin")
		if err != nil {
			log.Fatalln(err)
		}
	}

	fw, err := ioutil.ReadFile("../../modfw.bin")
	if err != nil {
		log.Fatalln(fw)
	}
	log.Println("test", jms.FlashInstallPatchAndBootFW(bootloader, fw))

	os.Exit(1)

	log.Println(jms.VersionGet())

	/*
		//log.Println(jms.CallFunction(0x3500, jmshal.CPUContext{DPTR: 1}))
		hooks, err := jms.HookGetAvailable()
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(hooks)
		log.Println(jms.HookCall(hooks[2], jmshal.CPUContext{DPTR: 1}))
	*/
	//	var buf [1024]byte
	//	log.Println(jms.CodeRead(0, buf[:]))
	//	log.Println(hex.EncodeToString(buf[:]))

	jms.XDATAWriteByte(0x714d, 0x2)

	flash, err := spiflash.New(jms.SPI, 512)
	log.Println(err)

	goToBootloader := func() {
		//
		log.Println(flash.ErasePage(0x0000))

		jms.ResetChip()

	}

	writeFw := func() {
		flashfw, err := ioutil.ReadFile("/home/bertold/vmshared/test2.bin")
		if err != nil {
			log.Fatalln(err)
		}
		start := time.Now()
		log.Println(flash.EraseChip())

		flash.Write(0x0e00, flashfw[:0x200])
		flash.Write(0x0000, flashfw[0x200:0x400])
		flash.Write(0x1000, flashfw[0x400:0xc400])
		flash.Write(0xd000, flashfw[0xc400:])
		jms.ResetChip()
		log.Println(time.Now().Sub(start))
	}

	if *bl {
		goToBootloader()
	} else {
		writeFw()
	}
	os.Exit(1)

	/*var page [65536]byte
	var page2 [65536]byte
	rand.Read(page[:])

	log.Println(flash.Write(0, page[:]))
	log.Println(flash.Read(0, page2[:]))

	log.Println("Full time", time.Now().Sub(start), bytes.Equal(page[:], page2[:]))*/

	//out[1] = 0
	//out[2] = 0
	//out[3] = 0
	//	var in [15]byte
	//	log.Println(jms.SpiHalfDuplex(out[:], in[:]), hex.EncodeToString(in[:]))

}
