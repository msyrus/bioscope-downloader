package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/grafov/m3u8"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/msyrus/bioscope-downloader/downloader"
	"github.com/msyrus/bioscope-downloader/playlist"
)

// func main() {
// 	msg := "I am Syrus, Minhaz Ahmed. I love"
// 	key, err := hex.DecodeString("b50057bdfd5639f192a5b701b1d6ff10")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	block, err := aes.NewCipher(key)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	dst := make([]byte, len(msg))
// 	iv := make([]byte, 16)
// 	cipher.NewCBCEncrypter(block, iv).CryptBlocks(dst, []byte(msg))
// 	println(string(dst))

// 	dct := make([]byte, len(msg))
// 	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dct, dst)
// 	println(string(dct))
// }

func main() {
	flag.Parse()
	id := flag.Arg(0)
	if id == "" {
		log.Fatal("item id is required")
	}

	// fetch master playlist
	plst, err := playlist.FetchMasterPlaylist(id)
	if err != nil {
		log.Fatal(err)
	}

	// Show variants
	lst := []*m3u8.Variant{}
	for i, v := range plst.Variants {
		if v != nil {
			lst = append(lst, v)
			fmt.Printf("%d. Resulation: %s,\tFrame Rate: %0.2f,\tCodecs: %s\n", i+1, v.Resolution, v.FrameRate, v.Codecs)
		}
	}
	if len(lst) == 0 {
		log.Fatalln("no variant found in master playlist")
	}

	sel := 0
	for {
		fmt.Print("\nSelect: ")
		if _, err = fmt.Scanln(&sel); err != nil {
			bufio.NewReader(os.Stdin).ReadLine()
		}
		if sel > 0 && sel <= len(lst) {
			break
		}
		fmt.Println("invalid input")
	}
	sel--

	mplst, err := playlist.FetchMediaPlaylist(lst[sel].URI)
	if err != nil {
		log.Fatal(err)
	}

	bar := pb.New64(downloader.Length(mplst)).SetUnits(pb.U_BYTES)
	bar.ShowPercent = true
	bar.ShowElapsedTime = true
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true

	name := id + "_" + strings.SplitN(lst[sel].Resolution, "x", 2)[1] + "p.mp4"
	f, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	bar.Set64(s.Size())
	bar.Start()
	if err = downloader.DownloadAfterBytes(io.MultiWriter(f, bar), mplst, s.Size()); err != nil {
		log.Fatal(err)
	}

	bar.FinishPrint("Completed")
}
