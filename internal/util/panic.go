package util

import "log"

func PanicIfErr(err error, tip ...string) {
	if err == nil {
		return
	}
	for _, v := range tip {
		log.Println(v)
	}
	log.Panic(err)
}

func FatalIfErr(err error, tip ...string) {
	if err == nil {
		return
	}
	for _, v := range tip {
		log.Println(v)
	}
	log.Fatal(err)
}
