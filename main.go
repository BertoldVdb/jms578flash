package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/BertoldVdb/jms578flash/jmshal"
	"github.com/BertoldVdb/jms578flash/jmsmods"
	"github.com/BertoldVdb/jms578flash/scsi"
)

func readFile(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}

	return os.ReadFile(path)
}

func main() {
	dev := flag.String("dev", "152d:0000", "Device to use")
	unsafe := flag.Bool("unsafe", false, "Allow writing to the flash memory")
	bootrom := flag.String("bootrom", "", "Path to dumped bootrom")
	firmware := flag.String("firmware", "", "Path to firmare")

	flash := flag.Bool("flash", false, "Flash given firmware to device")
	extract := flag.Bool("extract", false, "Read current firmware form device")
	dumprom := flag.Bool("dumprom", false, "Attempt to dump bootrom")

	boot := flag.Bool("boot", true, "Boot new firmware after flashing")
	dohook := flag.Bool("hook", true, "Attempt to add hooks to loaded firmware")
	mods := flag.String("mods", "", "Comma separated list of mods to add to the firmare")
	flag.Parse()

	actions := 0
	if *flash {
		actions++
	}
	if *extract {
		/* If we want to extract the firmware, flash should not be written, obviously */
		*unsafe = false
		actions++
	}
	if *dumprom {
		actions++
	}
	if actions != 1 {
		log.Fatalln("You can only specify one of '-flash','-extract' or '-dumprom'")
	}

	sdev, err := scsi.New(*dev)
	if err != nil {
		log.Fatalln(err)
	}

	jms, err := jmshal.New(sdev, *unsafe)
	if err != nil {
		log.Fatalln(err)
	}

	jms.LogFunc = log.Printf

	if *dumprom {
		if *bootrom == "" {
			log.Fatalln("Bootrom filename is missing")
		}

		log.Println("Trying to dump bootrom to", *bootrom)
		rom, err := jms.DumpBootrom()
		if err != nil {
			log.Fatalln("Dumping bootrom failed:", err)
		}
		if err := os.WriteFile(*bootrom, rom, 0644); err != nil {
			log.Fatalln("Failed to write to file:", err)
		}
		log.Println(len(rom), "bytes written to", *bootrom)
		return
	}

	rom, err := readFile(*bootrom)
	if err != nil {
		log.Fatalln("Failed to load specified bootrom:", err)
	}

	if *firmware == "" {
		log.Fatalln("Firmware filename is missing")
	}

	if *extract {
		if rom != nil {
			if err := jms.RebootToPatched(rom); err != nil {
				log.Println("Failed to access patched firmware:", err)
				log.Fatalln("You may want to remove the bootrom argument.")
			}
		}

		fw, err := jms.FlashReadFirmware()
		if err != nil {
			log.Fatalln("Failed to read firmware:", err)
		}

		if err := os.WriteFile(*firmware, fw, 0644); err != nil {
			log.Fatalln("Failed to write to file:", err)
		}
		log.Println(len(fw), "bytes written to", *firmware)
		return
	}

	if *flash {
		fw, err := os.ReadFile(*firmware)
		if err != nil {
			log.Fatalln("Failed to read firmware:", err)
		}
		var modjms []jmsmods.Mod
		if len(*mods) > 0 {
			for _, m := range strings.Split(*mods, ",") {
				modjms = append(modjms, jmsmods.Mod(m))
			}
		}

		if err := jms.FlashPatchWriteAndBootFW(rom, fw, *dohook, modjms, *boot); err != nil {
			log.Fatalln("Failed to write flash:", err)
		}
		log.Println("Flash writing complete")
	}
}
